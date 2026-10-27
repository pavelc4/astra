package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pavelc4/astra/internal/cache"
)

func TestPixivEndpoints_Validation(t *testing.T) {
	c := cache.New(10, time.Minute)
	defer c.Close()
	h := NewWithCache(c)

	t.Run("PixivSearch missing word", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/pixiv/search", nil)
		rec := httptest.NewRecorder()
		h.PixivSearch(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("PixivDownload missing url", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/pixiv/download", nil)
		rec := httptest.NewRecorder()
		h.PixivDownload(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("PixivImage missing url", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/pixiv/image", nil)
		rec := httptest.NewRecorder()
		h.PixivImage(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

func TestPixivRoutes_Registered(t *testing.T) {
	c := cache.New(10, time.Minute)
	defer c.Close()
	h := NewWithCache(c)

	r := chi.NewRouter()
	h.Register(r)

	// Test root endpoints listing
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Data struct {
			Endpoints []string `json:"endpoints"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode root response: %v", err)
	}

	expectedEndpoints := []string{
		"/api/pixiv/all",
		"/api/pixiv/ugoira",
		"/api/pixiv/novel",
		"/api/pixiv/user",
		"/api/pixiv/search",
		"/api/pixiv/ranking",
	}

	registered := make(map[string]bool)
	for _, ep := range resp.Data.Endpoints {
		registered[ep] = true
	}

	for _, ep := range expectedEndpoints {
		if !registered[ep] {
			t.Errorf("expected endpoint %q to be in root endpoints list", ep)
		}
	}

	// Test novel endpoint missing url via router
	novelReq := httptest.NewRequest(http.MethodGet, "/api/pixiv/novel", nil)
	novelRec := httptest.NewRecorder()
	r.ServeHTTP(novelRec, novelReq)
	if novelRec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for novel missing url, got %d", novelRec.Code)
	}

	// Test user endpoint missing url via router
	userReq := httptest.NewRequest(http.MethodGet, "/api/pixiv/user", nil)
	userRec := httptest.NewRecorder()
	r.ServeHTTP(userRec, userReq)
	if userRec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for user missing url, got %d", userRec.Code)
	}
}
