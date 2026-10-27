package handler

import (
	"fmt"
	"net/http"

	"github.com/pavelc4/astra/internal/cache"
	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/response"
)

var platformSuccessMsgs = map[string]string{
	"tiktok":     "TikTok media fetched successfully",
	"twitter":    "Twitter media fetched successfully",
	"instagram":  "Instagram media fetched successfully",
	"facebook":   "Facebook media fetched successfully",
	"threads":    "Threads media fetched successfully",
	"reddit":     "Reddit media fetched successfully",
	"pinterest":  "Pinterest media fetched successfully",
	"terabox":    "TeraBox media fetched successfully",
	"spotify":    "Spotify media fetched successfully",
	"soundcloud": "SoundCloud media fetched successfully",
	"capcut":     "CapCut media fetched successfully",
	"linkedin":   "LinkedIn media fetched successfully",
	"pixiv":      "Pixiv media fetched successfully",
}

// UniversalDownload automatically resolves the given URL to its respective platform
// extractor and returns standard media responses with multi-tier caching and singleflight.
func (h *Handlers) UniversalDownload(w http.ResponseWriter, r *http.Request) {
	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		response.HandleError(w, errors.NewValidation("url parameter is required"))
		return
	}

	nurl := cache.NormalizeURL(rawURL)
	extractor, err := h.registry.Resolve(nurl)
	if err != nil {
		response.HandleError(w, err)
		return
	}

	successMsg, ok := platformSuccessMsgs[extractor.Name()]
	if !ok {
		successMsg = fmt.Sprintf("%s media fetched successfully", extractor.Name())
	}
	key := cache.HashKey(successMsg, nurl)

	data, cachedErr, state := h.cache.Get(key)
	if state == cache.Fresh {
		if cachedErr != nil {
			response.HandleError(w, cachedErr)
			return
		}
		if target, ok := decodeCached[any](data); ok {
			response.OK(w, target, successMsg+" (cached)")
			return
		}
	}

	v, err, _ := h.sf.Do(key, func() (any, error) {
		return extractor.Extract(r.Context(), nurl)
	})

	if err != nil {
		if state == cache.Stale {
			if target, ok := decodeCached[any](data); ok {
				response.OK(w, target, successMsg+" (stale)")
				return
			}
		}
		h.cache.SetErr(key, err, h.negTTL)
		response.HandleError(w, err)
		return
	}

	h.cache.SetOK(key, v, h.freshTTL, h.staleTTL)
	response.OK(w, v, successMsg)
}
