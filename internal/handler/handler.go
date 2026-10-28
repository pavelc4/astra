package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/pavelc4/astra/internal/cache"
	"github.com/pavelc4/astra/internal/engine"
	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/response"
)

// Handlers holds the HTTP layer's injected dependencies. Constructed once at the
// composition root (main) and wired via Register — no package-level handler or
// cache globals.
type Handlers struct {
	cache    *cache.Cache
	registry *engine.Registry
	sf       singleflight.Group
	freshTTL time.Duration // cached result served directly
	staleTTL time.Duration // served as last-known-good if a refresh fails
	negTTL   time.Duration // failed fetch negatively cached this long
}

// New builds Handlers with a bounded persistent download cache. If CACHE_DB_PATH
// is defined in the environment, it uses that path; otherwise defaults to
// "astra_cache.db".
func New() *Handlers {
	dbPath := os.Getenv("CACHE_DB_PATH")
	if dbPath == "" {
		dbPath = "astra_cache.db"
	}
	// Max 500 items in RAM L1 (~1-2MB RAM), remainder on disk in SQLite L2. Ideal for 1GB RAM VPS.
	return NewWithCache(cache.New(500, 5*time.Minute, dbPath))
}

// NewWithCache builds Handlers with a custom cache instance, primarily for tests.
func NewWithCache(c *cache.Cache) *Handlers {
	return &Handlers{
		cache:    c,
		registry: engine.DefaultRegistry(),
		freshTTL: 24 * time.Hour, // Serve directly for 24h
		staleTTL: 72 * time.Hour, // Serve as stale fallback for 3 days
		negTTL:   10 * time.Second, // Transient error negative cache
	}
}

// WithRegistry sets a custom extractor registry on Handlers, useful for testing.
func (h *Handlers) WithRegistry(reg *engine.Registry) *Handlers {
	h.registry = reg
	return h
}

// normalizeURL is a backward-compatible wrapper around cache.NormalizeURL.
func normalizeURL(rawURL string) string {
	return cache.NormalizeURL(rawURL)
}

// decodeCached safely casts data to T or unmarshals []byte SQLite rows into T.
func decodeCached[T any](data any) (T, bool) {
	if d, ok := data.(T); ok {
		return d, true
	}
	if raw, ok := data.([]byte); ok {
		var target T
		if err := json.Unmarshal(raw, &target); err == nil {
			return target, true
		}
	}
	var zero T
	return zero, false
}

// download builds a caching download handler for a fetch function. Go doesn't
// allow generic methods, so the injected *Handlers is passed as the first
// argument to reach the cache and the singleflight group.
func download[T any](h *Handlers, fetchFunc func(ctx context.Context, url string) (T, error), successMsg string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rawURL := r.URL.Query().Get("url")
		if rawURL == "" {
			response.HandleError(w, errors.NewValidation("url parameter is required"))
			return
		}

		// Key namespaced by operation: the same URL served by different endpoints
		// (e.g. instagram download vs profile) returns different types T, so they
		// must not share a cache entry or a singleflight slot.
		nurl := cache.NormalizeURL(rawURL)
		key := cache.HashKey(successMsg, nurl)

		data, cachedErr, state := h.cache.Get(key)
		if state == cache.Fresh {
			if cachedErr != nil {
				response.HandleError(w, cachedErr)
				return
			}
			if target, ok := decodeCached[T](data); ok {
				response.OK(w, target, successMsg+" (cached)")
				return
			}
		}

		// singleflight collapses concurrent duplicate fetches for the same key
		// into one upstream call — the key defence against rate-limited scrape
		// targets when a link is requested by many clients at once.
		v, err, _ := h.sf.Do(key, func() (any, error) {
			return fetchFunc(r.Context(), nurl)
		})

		if err != nil {
			// Prefer last-known-good over a transient upstream failure (e.g. a
			// rate-limit 400): if this request saw a stale copy, serve it and
			// keep the stale entry (don't overwrite it with a negative one).
			if state == cache.Stale {
				if target, ok := decodeCached[T](data); ok {
					response.OK(w, target, successMsg+" (stale)")
					return
				}
			}
			h.cache.SetErr(key, err, h.negTTL)
			response.HandleError(w, err)
			return
		}

		if d, ok := v.(T); ok {
			// Redundant across singleflight followers (same data), but cheap and
			// keeps the write on the request path rather than in the shared fn.
			h.cache.SetOK(key, d, h.freshTTL, h.staleTTL)
			response.OK(w, d, successMsg)
			return
		}
		response.HandleError(w, errors.NewValidation("internal type mismatch"))
	}
}
