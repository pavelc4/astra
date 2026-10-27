package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pavelc4/astra/internal/cache"
	"github.com/pavelc4/astra/internal/engine"
	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/media"
	"github.com/pavelc4/astra/internal/response"
)

type mockExtractor struct {
	name    string
	calls   atomic.Int64
	matcher func(rawURL string) bool
	extract func(ctx context.Context, rawURL string) (any, error)
}

func (m *mockExtractor) Name() string { return m.name }
func (m *mockExtractor) Matches(rawURL string) bool {
	if m.matcher != nil {
		return m.matcher(rawURL)
	}
	return strings.Contains(rawURL, m.name)
}
func (m *mockExtractor) Extract(ctx context.Context, rawURL string) (any, error) {
	m.calls.Add(1)
	if m.extract != nil {
		return m.extract(ctx, rawURL)
	}
	return media.Downloads(m.name, "Mock Title", "https://example.com/thumb.jpg", nil), nil
}

func TestUniversalDownload_MissingURL(t *testing.T) {
	c := cache.New(10, time.Minute)
	defer c.Close()
	h := NewWithCache(c)

	req := httptest.NewRequest(http.MethodGet, "/api/download", nil)
	rec := httptest.NewRecorder()

	h.UniversalDownload(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", rec.Code)
	}

	var resp response.APIResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error == nil || resp.Error.Code != errors.CodeMissingParam {
		t.Errorf("expected MISSING_PARAMETER error, got %+v", resp.Error)
	}
}

func TestUniversalDownload_UnsupportedURL(t *testing.T) {
	c := cache.New(10, time.Minute)
	defer c.Close()
	h := NewWithCache(c)

	req := httptest.NewRequest(http.MethodGet, "/api/download?url=https://unknown-domain.example/xyz", nil)
	rec := httptest.NewRecorder()

	h.UniversalDownload(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity, got %d", rec.Code)
	}

	var resp response.APIResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error == nil || string(resp.Error.Code) != "INVALID_URL" {
		t.Errorf("expected INVALID_URL error, got %+v", resp.Error)
	}
}

func TestUniversalDownload_SuccessAndCaching(t *testing.T) {
	c := cache.New(10, time.Minute)
	defer c.Close()

	reg := engine.NewRegistry()
	mock := &mockExtractor{name: "mockplatform"}
	reg.Register(mock)

	h := NewWithCache(c).WithRegistry(reg)

	// First request - fresh extraction
	req1 := httptest.NewRequest(http.MethodGet, "/api/download?url=https://mockplatform.com/item/123", nil)
	rec1 := httptest.NewRecorder()
	h.UniversalDownload(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("first request: expected 200 OK, got %d (body: %s)", rec1.Code, rec1.Body.String())
	}
	if mock.calls.Load() != 1 {
		t.Fatalf("expected 1 extractor call, got %d", mock.calls.Load())
	}

	// Second request with tracking parameter - should hit cache
	req2 := httptest.NewRequest(http.MethodGet, "/api/download?url=https://www.mockplatform.com/item/123/?utm_source=test", nil)
	rec2 := httptest.NewRecorder()
	h.UniversalDownload(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("second request: expected 200 OK, got %d", rec2.Code)
	}
	if mock.calls.Load() != 1 {
		t.Fatalf("expected still 1 extractor call due to cache hit, got %d", mock.calls.Load())
	}

	var resp response.APIResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !strings.Contains(resp.Message, "(cached)") {
		t.Errorf("expected cached response message, got %s", resp.Message)
	}
}

func TestUniversalDownload_Singleflight(t *testing.T) {
	c := cache.New(10, time.Minute)
	defer c.Close()

	reg := engine.NewRegistry()
	mock := &mockExtractor{
		name: "slowplatform",
		extract: func(ctx context.Context, rawURL string) (any, error) {
			time.Sleep(30 * time.Millisecond)
			return media.Downloads("slowplatform", "Slow Title", "", nil), nil
		},
	}
	reg.Register(mock)

	h := NewWithCache(c).WithRegistry(reg)

	var wg sync.WaitGroup
	for i := 0; i < 15; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/api/download?url=https://slowplatform.com/video/456", nil)
			rec := httptest.NewRecorder()
			h.UniversalDownload(rec, req)
			if rec.Code != http.StatusOK {
				t.Errorf("concurrent request failed: %d", rec.Code)
			}
		}()
	}
	wg.Wait()

	if got := mock.calls.Load(); got != 1 {
		t.Fatalf("expected 1 extractor call across 15 parallel requests, got %d", got)
	}
}

func TestUniversalDownload_UpstreamError(t *testing.T) {
	c := cache.New(10, time.Minute)
	defer c.Close()

	reg := engine.NewRegistry()
	mock := &mockExtractor{
		name: "errorplatform",
		extract: func(ctx context.Context, rawURL string) (any, error) {
			return nil, errors.NewUpstream("mock upstream failure")
		},
	}
	reg.Register(mock)

	h := NewWithCache(c).WithRegistry(reg)

	req := httptest.NewRequest(http.MethodGet, "/api/download?url=https://errorplatform.com/err", nil)
	rec := httptest.NewRecorder()
	h.UniversalDownload(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 Bad Gateway, got %d", rec.Code)
	}
}
