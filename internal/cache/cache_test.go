package cache_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pavelc4/astra/internal/cache"
)

func TestFreshStaleMiss(t *testing.T) {
	c := cache.New(10, time.Hour) // in-memory
	defer c.Close()
	c.SetOK("k", "v", 20*time.Millisecond, 60*time.Millisecond)

	if d, _, st := c.Get("k"); st != cache.Fresh || d != "v" {
		t.Fatalf("want Fresh/v, got state=%v data=%v", st, d)
	}
	time.Sleep(35 * time.Millisecond)
	if d, _, st := c.Get("k"); st != cache.Stale || d != "v" {
		t.Fatalf("want Stale/v, got state=%v data=%v", st, d)
	}
	time.Sleep(35 * time.Millisecond)
	if _, _, st := c.Get("k"); st != cache.Miss {
		t.Fatalf("want Miss after stale window, got %v", st)
	}
}

func TestNegativeNeverStale(t *testing.T) {
	c := cache.New(10, time.Hour)
	defer c.Close()
	c.SetErr("k", errors.New("boom"), 20*time.Millisecond)

	if _, e, st := c.Get("k"); st != cache.Fresh || e == nil {
		t.Fatalf("want Fresh error, got state=%v err=%v", st, e)
	}
	time.Sleep(35 * time.Millisecond)
	if _, _, st := c.Get("k"); st != cache.Miss {
		t.Fatalf("expired negative entry must be Miss, got %v", st)
	}
}

func TestBounded(t *testing.T) {
	c := cache.New(3, time.Hour)
	defer c.Close()
	for i := 0; i < 20; i++ {
		c.SetOK(fmt.Sprintf("k%d", i), i, time.Hour, time.Hour)
	}
	// Verify stats
	hits, stale, misses := c.Stats()
	if hits != 0 || stale != 0 || misses != 0 {
		t.Errorf("unexpected stats: %d, %d, %d", hits, stale, misses)
	}
}

func TestSQLitePersistence_SurvivesRestart(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_cache.db")

	type SampleMedia struct {
		Title string `json:"title"`
		URL   string `json:"url"`
	}

	key := cache.HashKey("ig_download", "https://instagram.com/reel/123")
	media := SampleMedia{Title: "Cool Reel", URL: "https://cdn.example.com/reel.mp4"}

	// 1. Instance 1 writes to cache and shuts down
	c1 := cache.New(100, time.Hour, dbPath)
	c1.SetOK(key, media, 10*time.Minute, 1*time.Hour)
	_ = c1.Close()

	// 2. Instance 2 opens the same database (simulating server restart)
	c2 := cache.New(100, time.Hour, dbPath)
	defer c2.Close()

	data, err, state := c2.Get(key)
	if state != cache.Fresh {
		t.Fatalf("expected state Fresh after restart, got %v", state)
	}
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}

	raw, ok := data.([]byte)
	if !ok {
		t.Fatalf("expected []byte from SQLite tier, got %T", data)
	}
	if len(raw) == 0 {
		t.Fatalf("expected non-empty raw data")
	}

	// Verify hits stats
	hits, _, _ := c2.Stats()
	if hits != 1 {
		t.Errorf("expected 1 hit, got %d", hits)
	}
}

func TestSQLiteNegativeCache_SurvivesRestart(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_err_cache.db")

	key := cache.HashKey("fetch", "https://bad.url")

	// 1. Instance 1 sets error
	c1 := cache.New(100, time.Hour, dbPath)
	c1.SetErr(key, errors.New("upstream failed 502"), 10*time.Minute)
	_ = c1.Close()

	// 2. Instance 2 restarts and retrieves error
	c2 := cache.New(100, time.Hour, dbPath)
	defer c2.Close()

	_, cachedErr, state := c2.Get(key)
	if state != cache.Fresh {
		t.Fatalf("expected Fresh error, got %v", state)
	}
	if cachedErr == nil || cachedErr.Error() != "upstream failed 502" {
		t.Fatalf("unexpected cached error: %v", cachedErr)
	}
}

func TestSQLiteCleanupLoop(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_cleanup.db")

	c := cache.New(10, 20*time.Millisecond, dbPath)
	defer c.Close()

	key := "quick_expire"
	c.SetOK(key, "data", 10*time.Millisecond, 20*time.Millisecond)

	time.Sleep(50 * time.Millisecond)
	_, _, st := c.Get(key)
	if st != cache.Miss {
		t.Fatalf("expected Miss after expiration sweep, got %v", st)
	}
}

func BenchmarkCache_Get_MemoryL1(b *testing.B) {
	c := cache.New(1000, time.Hour)
	defer c.Close()
	key := "bench_key"
	c.SetOK(key, "bench_data", time.Hour, time.Hour)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, _ = c.Get(key)
	}
}

func BenchmarkCache_SetOK_Memory(b *testing.B) {
	c := cache.New(10000, time.Hour)
	defer c.Close()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		c.SetOK("k", "v", time.Hour, time.Hour)
	}
}

func BenchmarkCache_SetOK_SQLite(b *testing.B) {
	tempDir := b.TempDir()
	dbPath := filepath.Join(tempDir, "bench_sqlite.db")
	c := cache.New(10000, time.Hour, dbPath)
	defer c.Close()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		c.SetOK("k", "v", time.Hour, time.Hour)
	}
}

func BenchmarkCache_Parallel_Get_L1_RAM(b *testing.B) {
	c := cache.New(10000, time.Hour)
	defer c.Close()
	key := "hot_post_key"
	c.SetOK(key, map[string]string{"title": "Trending Reel", "url": "https://cdn.example.com/v.mp4"}, time.Hour, time.Hour)

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, _ = c.Get(key)
		}
	})
}

func BenchmarkCache_Parallel_Mixed_80Read_20Write_Memory(b *testing.B) {
	c := cache.New(10000, time.Hour)
	defer c.Close()
	for i := 0; i < 100; i++ {
		c.SetOK(fmt.Sprintf("k%d", i), "payload", time.Hour, time.Hour)
	}

	b.ResetTimer()
	b.ReportAllocs()
	var opCount atomic.Uint64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			n := opCount.Add(1)
			key := fmt.Sprintf("k%d", n%100)
			if n%5 == 0 { // 20% writes
				c.SetOK(key, "updated_payload", time.Hour, time.Hour)
			} else { // 80% reads
				_, _, _ = c.Get(key)
			}
		}
	})
}

func BenchmarkCache_Parallel_Mixed_80Read_20Write_SQLite(b *testing.B) {
	tempDir := b.TempDir()
	dbPath := filepath.Join(tempDir, "bench_mixed.db")
	c := cache.New(10000, time.Hour, dbPath)
	defer c.Close()

	for i := 0; i < 100; i++ {
		c.SetOK(fmt.Sprintf("k%d", i), "payload", time.Hour, time.Hour)
	}

	b.ResetTimer()
	b.ReportAllocs()
	var opCount atomic.Uint64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			n := opCount.Add(1)
			key := fmt.Sprintf("k%d", n%100)
			if n%5 == 0 { // 20% writes
				c.SetOK(key, "updated_payload", time.Hour, time.Hour)
			} else { // 80% reads
				_, _, _ = c.Get(key)
			}
		}
	})
}

func BenchmarkCache_ColdStart_SQLite_Recovery(b *testing.B) {
	tempDir := b.TempDir()
	dbPath := filepath.Join(tempDir, "bench_cold.db")

	// Pre-populate 500 keys in SQLite
	popCache := cache.New(1000, time.Hour, dbPath)
	for i := 0; i < 500; i++ {
		popCache.SetOK(fmt.Sprintf("cold_key_%d", i), "payload", time.Hour, time.Hour)
	}
	_ = popCache.Close()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// New instance simulating cold start
		c := cache.New(1000, time.Hour, dbPath)
		for k := 0; k < 50; k++ {
			_, _, _ = c.Get(fmt.Sprintf("cold_key_%d", k))
		}
		_ = c.Close()
	}
}

func BenchmarkCache_Eviction_Under_Pressure(b *testing.B) {
	// Small capacity (100) flooded with thousands of unique keys
	c := cache.New(100, time.Hour)
	defer c.Close()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		c.SetOK(fmt.Sprintf("flood_%d", i), "data", time.Hour, time.Hour)
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
