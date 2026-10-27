package terabox

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/httpclient"
	"github.com/pavelc4/astra/internal/media"
)

func FetchData(ctx context.Context, teraboxURL string) (*media.Media, error) {
	nonce, err := fetchNonce(ctx)
	if err != nil {
		return nil, err
	}

	form := url.Values{
		"action": {"terabox_fetch"},
		"url":    {teraboxURL},
		"nonce":  {nonce},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, AdminAjaxEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Origin", "https://teradownloaderz.com")
	req.Header.Set("Referer", "https://teradownloaderz.com/")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	for k, v := range httpclient.DefaultHeaders {
		req.Header[k] = v
	}

	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return nil, errors.NewUpstream(fmt.Sprintf("Terabox fetch failed: %s", err.Error()))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewUpstream("Terabox response read failed")
	}

	var apiResp struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, errors.NewUpstream(fmt.Sprintf("Failed to parse Terabox API response: %s", err.Error()))
	}

	if !apiResp.Success {
		var errMsg string
		_ = json.Unmarshal(apiResp.Data, &errMsg)
		if errMsg == "" {
			errMsg = string(body)
		}
		return nil, errors.NewUpstream(fmt.Sprintf("Terabox API security/fetch failed: %s", errMsg))
	}

	var data teraboxData
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		return nil, errors.NewUpstream(fmt.Sprintf("Failed to parse Terabox data block: %s", err.Error()))
	}

	if data.Status != "success" {
		return nil, errors.NewUpstream("Terabox API returned non-success status")
	}

	var downloads []media.Item
	var title string
	var thumbnail string

	for _, file := range data.Files {
		if title == "" {
			title = file.Name
		}
		if thumbnail == "" {
			thumbnail = file.ThumbnailURL
		}

		mediaType := media.Video
		switch file.FileType {
		case "video":
			mediaType = media.Video
		case "image":
			mediaType = media.Image
		case "audio":
			mediaType = media.Audio
		}

		if file.DownloadURL != "" {
			downloads = append(downloads, media.Item{
				Label:   file.Name,
				URL:     file.DownloadURL,
				Type:    mediaType,
				Quality: file.Quality,
			})
		}

		if len(file.StreamURLs) > 0 {
			var streamMap map[string]string
			if err := json.Unmarshal(file.StreamURLs, &streamMap); err == nil {
				for q, url := range streamMap {
					if url != "" {
						downloads = append(downloads, media.Item{
							Label:   fmt.Sprintf("%s (Stream %s)", file.Name, q),
							URL:     url,
							Type:    media.Video,
							Quality: q,
						})
					}
				}
			} else {
				var streamStr string
				if err := json.Unmarshal(file.StreamURLs, &streamStr); err == nil && streamStr != "" {
					downloads = append(downloads, media.Item{
						Label:   fmt.Sprintf("%s (Stream)", file.Name),
						URL:     streamStr,
						Type:    media.Video,
						Quality: file.Quality,
					})
				}
			}
		}
	}

	return media.Downloads("terabox", title, thumbnail, downloads), nil
}

func fetchNonce(ctx context.Context) (string, error) {
	bypassURL := fmt.Sprintf("https://teradownloaderz.com/?nowprocket=1&t=%d", time.Now().UnixNano())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, bypassURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	for k, v := range httpclient.DefaultHeaders {
		req.Header[k] = v
	}

	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return "", errors.NewUpstream(fmt.Sprintf("Terabox nonce fetch failed: %s", err.Error()))
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", errors.NewUpstream("Terabox nonce HTML parse failed")
	}

	script := doc.Find("#jquery-core-js-extra").Text()
	if script == "" {
		return "", errors.NewUpstream("Nonce script not found")
	}

	m := NonceRegex.FindStringSubmatch(script)
	if len(m) < 2 {
		return "", errors.NewUpstream("Nonce not found")
	}

	return m[1], nil
}
