package reddit

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/httpclient"
	"github.com/pavelc4/astra/internal/media"
)

func FetchData(ctx context.Context, targetURL string) (*media.Media, error) {
	// Normalize URL: strip query params and trailing slash for cleaner rapidsave request
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, errors.NewValidation(fmt.Sprintf("invalid URL: %s", err.Error()))
	}
	parsedURL.RawQuery = ""
	cleanURL := strings.TrimRight(parsedURL.String(), "/")

	rapidsaveURL := fmt.Sprintf("%s?url=%s", RapidSaveURL, url.QueryEscape(cleanURL))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rapidsaveURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Referer", RefererURL)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	for k, v := range httpclient.DefaultHeaders {
		req.Header[k] = v
	}

	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return nil, errors.NewUpstream(fmt.Sprintf("RapidSave request failed: %s", err.Error()))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewUpstream("RapidSave response read failed")
	}
	html := string(body)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, errors.NewUpstream("RapidSave HTML parse failed")
	}

	// Extract title from h2.text-center.text-truncate
	title := strings.TrimSpace(doc.Find("h2.text-center.text-truncate").First().Text())

	var downloads []media.Item
	var thumbnail string

	doc.Find("a.downloadbutton").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		switch {
		case extractVideoURL(href) != "":
			quality := parseQuality(extractVideoURL(href))
			downloads = append(downloads, media.Item{
				Label:   "Video " + quality,
				URL:     href,
				Type:    media.Video,
				Quality: quality,
			})
			if thumbnail == "" {
				if src, ok := doc.Find("img.img-fluid").First().Attr("src"); ok && src != "" {
					thumbnail = src
				}
			}

		case ImgReddRegex.MatchString(href):
			if thumbnail == "" {
				thumbnail = href
			}
			ext, mType := "image", media.Image
			if strings.HasSuffix(href, ".gif") {
				ext, mType = "gif", media.Video
			}
			downloads = append(downloads, media.Item{
				Label:   "Image " + ext,
				URL:     href,
				Type:    mType,
				Quality: "original",
			})

		case strings.Contains(href, "reddit.com/gallery/"):
			downloads = append(downloads, media.Item{
				Label:   "Gallery Link",
				URL:     href,
				Type:    media.Image,
				Quality: "gallery",
			})
		}
	})

	if len(downloads) == 0 {
		return nil, errors.NewUpstream("no downloadable media found in Reddit post")
	}

	return media.Downloads("reddit", title, thumbnail, downloads), nil
}

func parseQuality(videoURL string) string {
	lower := strings.ToLower(videoURL)
	switch {
	case strings.Contains(lower, "1080"):
		return "1080p"
	case strings.Contains(lower, "720"):
		return "720p"
	default:
		return "hd"
	}
}

func extractVideoURL(href string) string {
	if u, err := url.Parse(href); err == nil {
		if v := u.Query().Get("video_url"); v != "" {
			return v
		}
	}
	return VRedditRegex.FindString(href)
}
