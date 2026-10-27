package soundcloud

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/httpclient"
	"github.com/pavelc4/astra/internal/media"
)

func FetchData(ctx context.Context, targetURL string) (*media.Media, error) {
	encodedURL := base64.StdEncoding.EncodeToString([]byte(targetURL))
	hash := url.QueryEscape(encodedURL + HashSalt)
	payload := fmt.Sprintf("url=%s&token=%s&hash=%s", url.QueryEscape(targetURL), DefaultToken, hash)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, APIEndpoint, strings.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", "pll_language=en")
	req.Header.Set("Referer", RefererURL)
	for k, v := range httpclient.DefaultHeaders {
		req.Header[k] = v
	}

	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return nil, errors.NewUpstream(fmt.Sprintf("SoundCloud API request failed: %s", err.Error()))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewUpstream("SoundCloud response read failed")
	}

	var raw RawResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, errors.NewUpstream("failed to parse SoundCloud response")
	}

	var downloads []media.Item
	for _, m := range raw.Medias {
		if strings.HasSuffix(m.URL, ".json") {
			continue
		}

		mediaType := media.Audio
		if !m.AudioAvailable {
			mediaType = media.Video
		}

		downloads = append(downloads, media.Item{
			Label:   fmt.Sprintf("Audio %s (%s)", m.Quality, m.Extension),
			URL:     m.URL,
			Type:    mediaType,
			Quality: m.Quality,
		})
	}

	if len(downloads) == 0 {
		return nil, errors.NewUpstream("no downloadable media found in SoundCloud post")
	}

	return media.Downloads("soundcloud", raw.Title, raw.Thumbnail, downloads), nil
}
