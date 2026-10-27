package spotify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/httpclient"
	"github.com/pavelc4/astra/internal/media"
)

func FetchData(ctx context.Context, targetURL string) (*media.Media, error) {
	payload, _ := json.Marshal(map[string]string{"url": targetURL})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, MusicFabEndpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", RefererURL)
	req.Header.Set("Origin", OriginURL)
	for k, v := range httpclient.DefaultHeaders {
		req.Header[k] = v
	}

	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return nil, errors.NewUpstream(fmt.Sprintf("MusicFab API request failed: %s", err.Error()))
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, errors.NewUpstream(fmt.Sprintf("MusicFab API error: %d", resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewUpstream("MusicFab response read failed")
	}

	var raw SpotifyRawResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, errors.NewUpstream("failed to parse Spotify response")
	}

	meta := raw.Data.Metadata
	if meta.Download == "" {
		return nil, errors.NewUpstream("no downloadable media found in Spotify track")
	}

	downloads := []media.Item{
		{
			Label:   fmt.Sprintf("Audio (%s - %s)", meta.Artist, meta.Name),
			URL:     meta.Download,
			Type:    media.Audio,
			Quality: "320kbps",
		},
	}

	return media.Downloads("spotify", fmt.Sprintf("%s - %s", meta.Artist, meta.Name), meta.Image, downloads), nil
}
