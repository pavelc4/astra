// Package cache is a persistent two-tier (in-memory L1 + SQLite L2) cache
// for scrape responses.
//
// Entries carry two horizons: a fresh window (served directly) and a longer
// stale window (served only as last-known-good when a refresh fails). Memory
// is bounded so a flood of unique URLs can't grow the map without limit, while
// SQLite persists entries across server restarts.
package cache

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type State int

const (
	Miss  State = iota // absent or past the stale window
	Fresh              // within the fresh window — serve directly
	Stale              // fresh window passed, still serveable as last-known-good
)

type entry struct {
	data       any
	err        error
	freshUntil time.Time
	staleUntil time.Time
}

type Cache struct {
	mu    sync.RWMutex
	items map[string]entry
	max   int
	db    *sql.DB

	hits      atomic.Uint64
	staleHits atomic.Uint64
	misses    atomic.Uint64

	stopCleanup chan struct{}
}

// New builds a cache holding at most maxEntries in RAM. If an optional dbPath is
// provided and non-empty, entries are also persisted to SQLite.
func New(maxEntries int, cleanupInterval time.Duration, dbPath ...string) *Cache {
	c := &Cache{
		items:       make(map[string]entry, maxEntries),
		max:         maxEntries,
		stopCleanup: make(chan struct{}),
	}

	if len(dbPath) > 0 && dbPath[0] != "" {
		path := dbPath[0]
		dsn := path
		if !strings.Contains(dsn, "?") {
			dsn += "?_journal=WAL&_busy_timeout=5000"
		}
		db, err := sql.Open("sqlite3", dsn)
		if err != nil {
			slog.Error("failed to open sqlite cache, running memory-only", "path", path, "error", err)
		} else {
			if err := initSchema(db); err != nil {
				slog.Error("failed to init sqlite cache schema, running memory-only", "path", path, "error", err)
				_ = db.Close()
			} else {
				c.db = db
			}
		}
	}

	go c.cleanupLoop(cleanupInterval)
	return c
}

func initSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS cache_entries (
		key TEXT PRIMARY KEY,
		data BLOB,
		error TEXT,
		fresh_until INTEGER NOT NULL,
		stale_until INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_cache_stale ON cache_entries(stale_until);
	`
	_, err := db.Exec(schema)
	return err
}

// Get reports whether key is Fresh, Stale or Miss, returning the stored data/err
// for Fresh and Stale. First inspects L1 RAM; on miss, queries L2 SQLite.
func (c *Cache) Get(key string) (data any, err error, state State) {
	now := time.Now()

	// 1. Tier 1: In-memory lookup
	c.mu.RLock()
	e, ok := c.items[key]
	c.mu.RUnlock()

	if ok {
		switch {
		case now.Before(e.freshUntil):
			c.hits.Add(1)
			return e.data, e.err, Fresh
		case now.Before(e.staleUntil):
			c.staleHits.Add(1)
			return e.data, e.err, Stale
		}
	}

	// 2. Tier 2: SQLite lookup
	if c.db != nil {
		var (
			rawBytes  []byte
			errMsg    string
			freshUnix int64
			staleUnix int64
		)
		row := c.db.QueryRow(
			"SELECT data, error, fresh_until, stale_until FROM cache_entries WHERE key = ?",
			key,
		)
		if scanErr := row.Scan(&rawBytes, &errMsg, &freshUnix, &staleUnix); scanErr == nil {
			freshTime := time.Unix(freshUnix, 0)
			staleTime := time.Unix(staleUnix, 0)

			var cachedErr error
			if errMsg != "" {
				cachedErr = errors.New(errMsg)
			}

			var st State
			switch {
			case now.Before(freshTime):
				c.hits.Add(1)
				st = Fresh
			case now.Before(staleTime):
				c.staleHits.Add(1)
				st = Stale
			default:
				c.misses.Add(1)
				return nil, nil, Miss
			}

			// Warm L1 in-memory tier
			c.mu.Lock()
			if len(c.items) >= c.max {
				c.evictLocked()
			}
			c.items[key] = entry{
				data:       rawBytes,
				err:        cachedErr,
				freshUntil: freshTime,
				staleUntil: staleTime,
			}
			c.mu.Unlock()

			return rawBytes, cachedErr, st
		}
	}

	c.misses.Add(1)
	return nil, nil, Miss
}

// SetOK caches a successful result: fresh for freshTTL, then serveable-stale
// until staleTTL. Persists to both L1 RAM and L2 SQLite.
func (c *Cache) SetOK(key string, data any, freshTTL, staleTTL time.Duration) {
	now := time.Now()
	freshUntil := now.Add(freshTTL)
	staleUntil := now.Add(staleTTL)

	c.set(key, entry{
		data:       data,
		freshUntil: freshUntil,
		staleUntil: staleUntil,
	})

	if c.db != nil {
		raw, err := json.Marshal(data)
		if err == nil {
			query := `
			INSERT INTO cache_entries (key, data, error, fresh_until, stale_until)
			VALUES (?, ?, '', ?, ?)
			ON CONFLICT(key) DO UPDATE SET
				data = excluded.data,
				error = '',
				fresh_until = excluded.fresh_until,
				stale_until = excluded.stale_until;
			`
			if _, execErr := c.db.Exec(query, key, raw, freshUntil.Unix(), staleUntil.Unix()); execErr != nil {
				slog.Error("failed to write ok to sqlite cache", "key", key, "error", execErr)
			}
		}
	}
}

// SetErr negatively caches a failure.
func (c *Cache) SetErr(key string, err error, ttl time.Duration) {
	now := time.Now()
	until := now.Add(ttl)

	c.set(key, entry{
		err:        err,
		freshUntil: until,
		staleUntil: until,
	})

	if c.db != nil {
		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		}
		query := `
		INSERT INTO cache_entries (key, data, error, fresh_until, stale_until)
		VALUES (?, NULL, ?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET
			data = NULL,
			error = excluded.error,
			fresh_until = excluded.fresh_until,
			stale_until = excluded.stale_until;
		`
		if _, execErr := c.db.Exec(query, key, errMsg, until.Unix(), until.Unix()); execErr != nil {
			slog.Error("failed to write err to sqlite cache", "key", key, "error", execErr)
		}
	}
}

func (c *Cache) set(key string, e entry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.items[key]; !exists && len(c.items) >= c.max {
		c.evictLocked()
	}
	c.items[key] = e
}

func (c *Cache) evictLocked() {
	now := time.Now()
	for k, v := range c.items {
		if now.After(v.staleUntil) {
			delete(c.items, k)
		}
	}
	if len(c.items) < c.max {
		return
	}
	var oldestKey string
	var oldest time.Time
	for k, v := range c.items {
		if oldestKey == "" || v.staleUntil.Before(oldest) {
			oldestKey, oldest = k, v.staleUntil
		}
	}
	delete(c.items, oldestKey)
}

// Stats returns cumulative fresh hits, stale hits and misses.
func (c *Cache) Stats() (hits, stale, misses uint64) {
	return c.hits.Load(), c.staleHits.Load(), c.misses.Load()
}

// Close stops the cleanup loop and closes the SQLite database connection.
func (c *Cache) Close() error {
	select {
	case <-c.stopCleanup:
	default:
		close(c.stopCleanup)
	}
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

func (c *Cache) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCleanup:
			return
		case <-ticker.C:
			now := time.Now()
			c.mu.Lock()
			for k, v := range c.items {
				if now.After(v.staleUntil) {
					delete(c.items, k)
				}
			}
			c.mu.Unlock()

			if c.db != nil {
				if _, err := c.db.Exec("DELETE FROM cache_entries WHERE stale_until < ?", now.Unix()); err != nil {
					slog.Error("failed to sweep expired sqlite cache entries", "error", err)
				}
			}
		}
	}
}
