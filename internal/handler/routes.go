package handler

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/pavelc4/astra/internal/cache"
	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/platform/capcut"
	"github.com/pavelc4/astra/internal/platform/linkedin"
	"github.com/pavelc4/astra/internal/platform/meta/facebook"
	"github.com/pavelc4/astra/internal/platform/meta/instagram"
	"github.com/pavelc4/astra/internal/platform/meta/threads"
	"github.com/pavelc4/astra/internal/platform/pinterest"
	"github.com/pavelc4/astra/internal/platform/pixiv"
	"github.com/pavelc4/astra/internal/platform/reddit"
	"github.com/pavelc4/astra/internal/platform/soundcloud"
	"github.com/pavelc4/astra/internal/platform/spotify"
	"github.com/pavelc4/astra/internal/platform/terabox"
	"github.com/pavelc4/astra/internal/platform/tiktok"
	"github.com/pavelc4/astra/internal/platform/twitter"
	"github.com/pavelc4/astra/internal/response"
)

// Register wires every route onto r. All download endpoints share the injected
// request cache via download(h, ...); this is the single place platform routes
// live (replacing the former one-file-per-platform handlers).
func (h *Handlers) Register(r chi.Router) {
	r.Get("/health", h.Health)
	r.Get("/api/download", h.UniversalDownload)

	r.Route("/api/tiktok", func(r chi.Router) {
		r.Get("/download", download(h, tiktok.FetchData, "TikTok media fetched successfully"))
		r.Get("/profile", download(h, tiktok.FetchUser, "TikTok profile fetched successfully"))
		r.Get("/music", download(h, tiktok.FetchMusic, "TikTok music fetched successfully"))
	})
	r.Route("/api/twitter", func(r chi.Router) {
		r.Get("/download", download(h, twitter.FetchData, "Twitter media fetched successfully"))
	})
	r.Route("/api/meta", func(r chi.Router) {
		r.Get("/instagram/download", download(h, instagram.FetchData, "Instagram media fetched successfully"))
		r.Get("/instagram/profile", download(h, instagram.FetchProfileByURL, "Instagram profile fetched successfully"))
		r.Get("/instagram/stories", h.InstagramStories)
		r.Get("/facebook/download", download(h, facebook.FetchData, "Facebook media fetched successfully"))
		r.Get("/threads/download", download(h, threads.FetchData, "Threads media fetched successfully"))
	})
	r.Route("/api/reddit", func(r chi.Router) {
		r.Get("/download", download(h, reddit.FetchData, "Reddit media fetched successfully"))
	})
	r.Route("/api/pinterest", func(r chi.Router) {
		r.Get("/download", download(h, pinterest.FetchData, "Pinterest media fetched successfully"))
	})
	r.Route("/api/terabox", func(r chi.Router) {
		r.Get("/download", download(h, terabox.FetchData, "TeraBox media fetched successfully"))
	})
	r.Route("/api/spotify", func(r chi.Router) {
		r.Get("/download", download(h, spotify.FetchData, "Spotify media fetched successfully"))
	})
	r.Route("/api/soundcloud", func(r chi.Router) {
		r.Get("/download", download(h, soundcloud.FetchData, "SoundCloud media fetched successfully"))
	})
	r.Route("/api/capcut", func(r chi.Router) {
		r.Get("/download", download(h, capcut.FetchData, "CapCut media fetched successfully"))
	})
	r.Route("/api/linkedin", func(r chi.Router) {
		r.Get("/download", download(h, linkedin.FetchData, "LinkedIn media fetched successfully"))
	})
	r.Route("/api/pixiv", func(r chi.Router) {
		r.Get("/download", h.PixivDownload)
		r.Get("/profile", download(h, pixiv.FetchUserProfile, "Pixiv profile fetched successfully"))
		r.Get("/illustrations", download(h, pixiv.FetchUserIllustrations, "Pixiv illustrations fetched successfully"))
		r.Get("/bookmarks", download(h, pixiv.FetchUserBookmarks, "Pixiv bookmarks fetched successfully"))
		r.Get("/all", download(h, pixiv.FetchUserAllWorks, "Pixiv user works fetched successfully"))
		r.Get("/ugoira", download(h, pixiv.FetchUgoiraDirect, "Pixiv ugoira meta fetched successfully"))
		r.Get("/ranking", h.PixivRanking)
		r.Get("/image", h.PixivImage)
	})

	r.Get("/", handleRoot)
}

// InstagramStories is a bespoke handler: stories are fetched by username, not by
// the shared url→media flow, so it doesn't go through download().
func (h *Handlers) InstagramStories(w http.ResponseWriter, r *http.Request) {
	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		response.HandleError(w, errors.NewValidation("url parameter is required"))
		return
	}

	username := instagram.ExtractUsername(rawURL)
	if username == "" {
		response.HandleError(w, errors.NewValidation("could not extract username from URL"))
		return
	}

	data, err := instagram.FetchStories(r.Context(), username)
	if err != nil {
		response.HandleError(w, err)
		return
	}

	response.OK(w, data, "Instagram stories fetched successfully")
}

// PixivDownload returns the artwork metadata like any downloader, but rewrites
// every i.pximg.net URL (thumbnail + each download) into our own /image proxy
// endpoint — so the returned urls open/download directly without Pixiv's 403.
func (h *Handlers) PixivDownload(w http.ResponseWriter, r *http.Request) {
	targetURL := r.URL.Query().Get("url")
	if targetURL == "" {
		response.HandleError(w, errors.NewValidation("url parameter is required"))
		return
	}

	md, err := pixiv.FetchData(r.Context(), targetURL)
	if err != nil {
		response.HandleError(w, err)
		return
	}

	proxy := pixivProxyURL(r)
	md.Thumbnail = proxy(md.Thumbnail)
	for i := range md.Items {
		md.Items[i].URL = proxy(md.Items[i].URL)
		md.Items[i].Thumbnail = proxy(md.Items[i].Thumbnail)
	}

	response.OK(w, md, "Pixiv media fetched successfully")
}

// pixivProxyURL builds a closure that turns an i.pximg.net URL into our own
// /api/pixiv/image endpoint on the same origin as the incoming request.
func pixivProxyURL(r *http.Request) func(string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	base := scheme + "://" + r.Host + "/api/pixiv/image?url="
	return func(u string) string {
		if u == "" || !strings.Contains(u, "i.pximg.net") {
			return u
		}
		return base + url.QueryEscape(u)
	}
}

// PixivImage streams a Pixiv media file through the server so clients that
// can't set a Referer header can still fetch the bytes. Accepts either an
// artwork URL (e.g. /artworks/{id}) or a raw i.pximg.net URL directly.
func (h *Handlers) PixivImage(w http.ResponseWriter, r *http.Request) {
	targetURL := r.URL.Query().Get("url")
	if targetURL == "" {
		response.HandleError(w, errors.NewValidation("url parameter is required"))
		return
	}

	// Set before streaming: headers flush once the first bytes hit the wire.
	w.Header().Set("Content-Disposition", "attachment; filename=\"pixiv-download\"")
	ctype, err := pixiv.StreamDownload(r.Context(), targetURL, w)
	if err != nil {
		response.HandleError(w, err)
		return
	}

	w.Header().Set("Content-Type", ctype)
}

// PixivRanking fetches artwork rankings by mode and page (mode=daily|weekly|monthly|rookie|original|male|female|daily_r18).
func (h *Handlers) PixivRanking(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "daily"
	}
	pageStr := r.URL.Query().Get("page")
	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	key := cache.HashKey("pixiv_ranking_"+mode, strconv.Itoa(page))
	data, cachedErr, state := h.cache.Get(key)
	if state == cache.Fresh {
		if cachedErr != nil {
			response.HandleError(w, cachedErr)
			return
		}
		if target, ok := decodeCached[[]pixiv.RankingItem](data); ok {
			response.OK(w, target, "Pixiv rankings fetched successfully (cached)")
			return
		}
	}

	v, err, _ := h.sf.Do(key, func() (any, error) {
		return pixiv.FetchRankings(r.Context(), mode, page)
	})

	if err != nil {
		if state == cache.Stale {
			if target, ok := decodeCached[[]pixiv.RankingItem](data); ok {
				response.OK(w, target, "Pixiv rankings fetched successfully (stale)")
				return
			}
		}
		h.cache.SetErr(key, err, h.negTTL)
		response.HandleError(w, err)
		return
	}

	h.cache.SetOK(key, v, h.freshTTL, h.staleTTL)
	response.OK(w, v, "Pixiv rankings fetched successfully")
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	response.OK(w, map[string]any{
		"endpoints": []string{
			"/health",
			"/api/tiktok/download",
			"/api/tiktok/profile",
			"/api/tiktok/music",
			"/api/twitter/download",
			"/api/meta/instagram/download",
			"/api/meta/instagram/profile",
			"/api/meta/instagram/stories",
			"/api/meta/facebook/download",
			"/api/meta/threads/download",
			"/api/reddit/download",
			"/api/pinterest/download",
			"/api/terabox/download",
			"/api/spotify/download",
			"/api/soundcloud/download",
			"/api/capcut/download",
			"/api/linkedin/download",
			"/api/pixiv/download",
			"/api/pixiv/profile",
			"/api/pixiv/illustrations",
			"/api/pixiv/bookmarks",
			"/api/pixiv/image",
		},
	}, "Universal Downloader API is running")
}
