package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pavelc4/astra/internal/cache"
	"github.com/pavelc4/astra/internal/response"
)

// TestDownloadSingleflight proves concurrent duplicate requests collapse into a
// single upstream fetch — the defence against rate-limited scrape targets.
func TestDownloadSingleflight(t *testing.T) {
	c := cache.New(100, time.Hour) // in-memory
	defer c.Close()
	h := NewWithCache(c)

	var calls atomic.Int64
	slow := func(ctx context.Context, url string) (string, error) {
		calls.Add(1)
		time.Sleep(50 * time.Millisecond) // hold the flight open so others coalesce
		return "ok", nil
	}
	hf := download(h, slow, "msg")

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/?url=http://example.com/a", nil)
			hf(httptest.NewRecorder(), req)
		}()
	}
	wg.Wait()

	if got := calls.Load(); got != 1 {
		t.Fatalf("singleflight: want 1 upstream call for 20 concurrent identical requests, got %d", got)
	}
}

// TestDownloadCacheHit confirms a second request is served from cache (no second fetch).
func TestDownloadCacheHit(t *testing.T) {
	c := cache.New(100, time.Hour)
	defer c.Close()
	h := NewWithCache(c)

	var calls atomic.Int64
	fetch := func(ctx context.Context, url string) (string, error) {
		calls.Add(1)
		return "ok", nil
	}
	hf := download(h, fetch, "msg")

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/?url=http://example.com/b", nil)
		hf(httptest.NewRecorder(), req)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("cache: want 1 fetch for 3 sequential identical requests, got %d", got)
	}
}

// TestDownload_URLNormalizationCacheHit proves that URL variations with tracking parameters
// or trailing slashes hit the exact same cache entry instead of triggering a re-scrape.
func TestDownload_URLNormalizationCacheHit(t *testing.T) {
	c := cache.New(100, time.Hour)
	defer c.Close()
	h := NewWithCache(c)

	var calls atomic.Int64
	fetch := func(ctx context.Context, url string) (string, error) {
		calls.Add(1)
		return "scraped_content", nil
	}
	hf := download(h, fetch, "ig_download")

	urlVariants := []string{
		"https://www.instagram.com/reel/C7xyz123/?igsh=abcdef123&utm_source=copy",
		"http://instagram.com/reel/C7xyz123?utm_medium=social#tag",
		"https://instagram.com/reel/C7xyz123/",
		"https://instagram.com/reel/C7xyz123",
	}

	for _, u := range urlVariants {
		req := httptest.NewRequest("GET", "/?url="+u, nil)
		rec := httptest.NewRecorder()
		hf(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
	}

	if got := calls.Load(); got != 1 {
		t.Fatalf("expected exactly 1 upstream fetch for URL variations of the same content, got %d", got)
	}
}

// TestDownload_SQLiteRestoration confirms that after restarting the server with SQLite cache,
// cached objects are seamlessly deserialized into the expected target struct.
func TestDownload_SQLiteRestoration(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "handler_test.db")

	type MediaItem struct {
		Title string `json:"title"`
		URL   string `json:"url"`
	}

	rawTargetURL := "https://tiktok.com/@user/video/123456789"

	// 1. First server session writes to cache
	c1 := cache.New(100, time.Hour, dbPath)
	h1 := NewWithCache(c1)
	fetchFunc := func(ctx context.Context, u string) (MediaItem, error) {
		return MediaItem{Title: "TikTok Dance", URL: "https://cdn.example.com/dance.mp4"}, nil
	}

	hf1 := download(h1, fetchFunc, "tiktok_download")
	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/?url="+rawTargetURL, nil)
	hf1(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("first request failed with code %d", rec1.Code)
	}
	_ = c1.Close()

	// 2. Second server session starts (simulating app restart)
	c2 := cache.New(100, time.Hour, dbPath)
	defer c2.Close()
	h2 := NewWithCache(c2)

	var callsAfterRestart atomic.Int64
	fetchAfterRestart := func(ctx context.Context, u string) (MediaItem, error) {
		callsAfterRestart.Add(1)
		return MediaItem{}, nil
	}

	hf2 := download(h2, fetchAfterRestart, "tiktok_download")
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/?url="+rawTargetURL, nil)
	hf2(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on cached response after restart, got %d", rec2.Code)
	}

	var res response.APIResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if res.Message != "tiktok_download (cached)" {
		t.Errorf("expected message %q, got %q", "tiktok_download (cached)", res.Message)
	}

	if got := callsAfterRestart.Load(); got != 0 {
		t.Fatalf("expected 0 upstream calls after restart due to SQLite cache hit, got %d", got)
	}
}
