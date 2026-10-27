package linkedin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/httpclient"
	"github.com/pavelc4/astra/internal/media"
)

func FetchData(ctx context.Context, targetURL string) (*Result, error) {
	// 1. Fetch direct HTML to extract title & thumbnail (image)
	reqHTML, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create HTML request: %w", err)
	}
	// Use search engine bot User-Agent for clean SSR response from LinkedIn
	reqHTML.Header.Set("User-Agent", GooglebotUA)

	respHTML, err := httpclient.Client.Do(reqHTML)
	var title, thumbnail string
	if err == nil {
		defer respHTML.Body.Close()
		doc, errDoc := goquery.NewDocumentFromReader(respHTML.Body)
		if errDoc == nil {
			doc.Find("meta").Each(func(_ int, s *goquery.Selection) {
				property, _ := s.Attr("property")
				content, _ := s.Attr("content")
				if property == "og:title" {
					title = html.UnescapeString(content)
				} else if property == "og:image" {
					thumbnail = content
				}
			})
		}
	}

	// 2. Fetch video from saywhat.ai API
	payload, _ := json.Marshal(map[string]string{"url": targetURL})
	reqAPI, err := http.NewRequestWithContext(ctx, http.MethodPost, SayWhatEndpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create API request: %w", err)
	}

	reqAPI.Header.Set("Content-Type", "application/json")
	reqAPI.Header.Set("Referer", SayWhatReferer)
	for k, v := range httpclient.DefaultHeaders {
		reqAPI.Header[k] = v
	}

	respAPI, err := httpclient.Client.Do(reqAPI)
	var sayWhat SayWhatResponse
	if err == nil {
		defer respAPI.Body.Close()
		body, errRead := io.ReadAll(respAPI.Body)
		if errRead == nil {
			_ = json.Unmarshal(body, &sayWhat)
		}
	}

	var downloads []media.Item

	// 3. Map videos if found
	for _, v := range sayWhat.Videos {
		if v.URL != "" {
			downloads = append(downloads, media.Item{
				Label:   fmt.Sprintf("Video (%s)", v.Quality),
				URL:     v.URL,
				Type:    media.Video,
				Quality: v.Quality,
			})
		}
	}

	// 4. Fallback to high-res image if no videos are found
	if len(downloads) == 0 && thumbnail != "" {
		downloads = append(downloads, media.Item{
			Label: "Image (Original)",
			URL:   thumbnail,
			Type:  media.Image,
		})
	}

	if len(downloads) == 0 {
		return nil, errors.NewUpstream("no downloadable media found in LinkedIn post")
	}

	// Clean title suffix if present
	title = strings.TrimSuffix(title, " | LinkedIn")

	return &Result{
		Platform:  "linkedin",
		Title:     title,
		Thumbnail: thumbnail,
		Downloads: downloads,
	}, nil
}
