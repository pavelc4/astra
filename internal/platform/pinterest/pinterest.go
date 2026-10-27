package pinterest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/httpclient"
	"github.com/pavelc4/astra/internal/media"
)

// pinterestClient is a custom http.Client configured to follow redirects for pin.it URLs.
var pinterestClient = &http.Client{
	Timeout:   10 * time.Second,
	Transport: httpclient.Client.Transport,
}

func FetchData(ctx context.Context, targetURL string) (*media.Media, error) {
	// 1. Resolve final URL to extract Pin ID
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := pinterestClient.Do(req)
	if err != nil {
		return nil, errors.NewUpstream(fmt.Sprintf("failed to resolve URL: %s", err.Error()))
	}
	defer resp.Body.Close()

	finalURL := resp.Request.URL.String()
	matches := PinIDRegex.FindStringSubmatch(finalURL)
	if len(matches) < 2 {
		return nil, errors.NewValidation(fmt.Sprintf("could not extract Pin ID from URL: %s", finalURL))
	}
	pinID := matches[1]

	// 2. Fetch detailed metadata from Pinterest internal resource API
	apiURL := "https://www.pinterest.com/resource/PinResource/get/"
	dataParam, err := json.Marshal(map[string]interface{}{
		"options": map[string]interface{}{
			"id":            pinID,
			"field_set_key": "detailed",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request options: %w", err)
	}

	params := url.Values{}
	params.Set("data", string(dataParam))
	fullAPIURL := apiURL + "?" + params.Encode()

	apiReq, err := http.NewRequestWithContext(ctx, http.MethodGet, fullAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create API request: %w", err)
	}

	apiReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	apiReq.Header.Set("X-Pinterest-PWS-Handler", "www/[username].js")
	apiReq.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	apiReq.Header.Set("X-Requested-With", "XMLHttpRequest")

	apiResp, err := httpclient.Client.Do(apiReq)
	if err != nil {
		return nil, errors.NewUpstream(fmt.Sprintf("Pinterest API request failed: %s", err.Error()))
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode != http.StatusOK {
		return nil, errors.NewUpstream(fmt.Sprintf("Pinterest API returned status: %d", apiResp.StatusCode))
	}

	body, err := io.ReadAll(apiResp.Body)
	if err != nil {
		return nil, errors.NewUpstream("failed to read Pinterest API response body")
	}

	var parsed struct {
		ResourceResponse struct {
			Data map[string]interface{} `json:"data"`
		} `json:"resource_response"`
	}

	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, errors.NewUpstream("failed to parse Pinterest API JSON response")
	}

	data := parsed.ResourceResponse.Data
	if data == nil {
		return nil, errors.NewUpstream("Pinterest API returned empty data object")
	}

	// 3. Extract title, description, and thumbnail
	title := ""
	if t, ok := data["title"].(string); ok && t != "" {
		title = t
	} else if gt, ok := data["grid_title"].(string); ok && gt != "" {
		title = gt
	} else if desc, ok := data["description"].(string); ok && desc != "" {
		title = desc
	}

	thumbnail := ""
	if images, ok := data["images"].(map[string]interface{}); ok {
		if orig, ok := images["orig"].(map[string]interface{}); ok {
			if u, ok := orig["url"].(string); ok {
				thumbnail = u
			}
		} else if size736, ok := images["736x"].(map[string]interface{}); ok {
			if u, ok := size736["url"].(string); ok {
				thumbnail = u
			}
		}
	}

	var downloads []media.Item

	// parseMasterM3U8 fetches the master m3u8 playlist and returns (suffix→label) pairs ordered by resolution descending.
	parseMasterM3U8 := func(m3u8URL string) []struct{ Suffix, Label string } {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, m3u8URL, nil)
		if err != nil {
			return nil
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		resp, err := pinterestClient.Do(req)
		if err != nil {
			return nil
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil
		}
		matches := StreamEntryRe.FindAllStringSubmatch(string(body), -1)
		type entry struct {
			width  int
			suffix string
			label  string
		}
		var entries []entry
		seen := map[string]bool{}
		for _, m := range matches {
			res := m[1] // e.g. "720x1280"
			suffix := m[3] // e.g. "720w"
			if seen[suffix] {
				continue
			}
			seen[suffix] = true
			var w int
			fmt.Sscanf(res, "%dx", &w)
			label := fmt.Sprintf("%dp", w)
			entries = append(entries, entry{w, suffix, label})
		}
		// Sort descending by width
		for i := 0; i < len(entries); i++ {
			for j := i + 1; j < len(entries); j++ {
				if entries[j].width > entries[i].width {
					entries[i], entries[j] = entries[j], entries[i]
				}
			}
		}
		var result []struct{ Suffix, Label string }
		for _, e := range entries {
			result = append(result, struct{ Suffix, Label string }{e.suffix, e.label})
		}
		return result
	}

	// extractVideoDownloads converts a video_list map into direct MP4 DownloadItems.
	// It fetches the m3u8 playlist to detect actual available resolutions.
	extractVideoDownloads := func(videoList map[string]interface{}) []media.Item {
		var items []media.Item
		for _, val := range videoList {
			vMap, ok := val.(map[string]interface{})
			if !ok {
				continue
			}
			u, ok := vMap["url"].(string)
			if !ok || u == "" {
				continue
			}
			// Try to extract signature and build direct mp4 URLs
			matches := M3U8SigRegex.FindStringSubmatch(u)
			if len(matches) == 5 {
				sig := matches[4]
				p1, p2, p3 := matches[1], matches[2], matches[3]
				baseURL := fmt.Sprintf("https://v1.pinimg.com/videos/iht/hls/%s/%s/%s", p1, p2, p3)
				// Parse actual resolutions from master playlist
				qualities := parseMasterM3U8(u)
				if len(qualities) == 0 {
					// Fallback if m3u8 parse fails
					qualities = []struct{ Suffix, Label string }{
						{"720w", "720p"}, {"540w", "540p"}, {"360w", "360p"}, {"240w", "240p"},
					}
				}
				for _, q := range qualities {
					directURL := fmt.Sprintf("%s/%s_%s.cmfv", baseURL, sig, q.Suffix)
					items = append(items, media.Item{
						Label:   "Video " + q.Label,
						URL:     directURL,
						Type:    media.Video,
						Quality: q.Label,
					})
				}
				return items // return after first successful signature extraction
			}
			// Fallback: return the m3u8 as-is if no signature found
			items = append(items, media.Item{
				Label:   "Video HLS",
				URL:     u,
				Type:    media.Video,
				Quality: "hls",
			})
			return items
		}
		return items
	}

	// 4. Extract video streams (if present at root level)
	videoFound := false
	if videos, ok := data["videos"].(map[string]interface{}); ok && videos != nil {
		if videoList, ok := videos["video_list"].(map[string]interface{}); ok {
			items := extractVideoDownloads(videoList)
			if len(items) > 0 {
				downloads = append(downloads, items...)
				videoFound = true
			}
		}
	}

	// 5. Look for video inside story_pin_data if not found at root level
	if !videoFound {
		if story, ok := data["story_pin_data"].(map[string]interface{}); ok {
			if pages, ok := story["pages"].([]interface{}); ok {
				for _, page := range pages {
					if pMap, ok := page.(map[string]interface{}); ok {
						if blocks, ok := pMap["blocks"].([]interface{}); ok {
							for _, block := range blocks {
								if bMap, ok := block.(map[string]interface{}); ok {
									if videoObj, ok := bMap["video"].(map[string]interface{}); ok {
										if videoList, ok := videoObj["video_list"].(map[string]interface{}); ok {
											items := extractVideoDownloads(videoList)
											if len(items) > 0 {
												downloads = append(downloads, items...)
												videoFound = true
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// 6. If no video found, add the original image as a download option
	if !videoFound && thumbnail != "" {
		downloads = append(downloads, media.Item{
			Label:   "Image original",
			URL:     thumbnail,
			Type:    media.Image,
			Quality: "original",
		})
	}

	return media.Downloads("pinterest", title, thumbnail, downloads), nil
}
