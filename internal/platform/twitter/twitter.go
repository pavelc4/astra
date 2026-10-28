package twitter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/httpclient"
	"github.com/pavelc4/astra/internal/media"
	"github.com/pavelc4/astra/internal/resolution"
)

func FetchData(ctx context.Context, tweetURL string) (*Result, error) {
	form := url.Values{"q": {tweetURL}, "lang": {"en"}, "cftoken": {""}}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, SaveTwitterEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Origin", OriginURL)
	req.Header.Set("Referer", RefererURL)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	for k, v := range httpclient.DefaultHeaders {
		req.Header[k] = v
	}

	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return nil, errors.NewUpstream(fmt.Sprintf("SaveTwitter request failed: %s", err.Error()))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewUpstream("SaveTwitter response read failed")
	}

	var ajaxResp AjaxResponse
	if err := json.Unmarshal(body, &ajaxResp); err != nil {
		return nil, errors.NewUpstream("failed to parse SaveTwitter JSON response")
	}

	return ParseResult([]byte(ajaxResp.Data)), nil
}

func ParseResult(data []byte) *Result {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(data)))
	if err != nil {
		return &Result{Platform: "twitter", Type: "photo"}
	}

	var tweetID *string
	if v, ok := doc.Find("#TwitterId").Attr("value"); ok {
		tweetID = &v
	}

	var title *string
	if t := strings.TrimSpace(doc.Find(".tw-middle h3").First().Text()); t != "" {
		title = &t
	}

	var duration *string
	if d := strings.TrimSpace(doc.Find(".tw-middle p").First().Text()); d != "" {
		duration = &d
	}

	var thumbnail *string
	if src, ok := doc.Find(".thumbnail img").Attr("src"); ok {
		thumbnail = &src
	}
	if thumbnail == nil {
		if src, ok := doc.Find(".download-items__thumb img").Attr("src"); ok {
			thumbnail = &src
		}
	}

	var downloads []media.Item

	twVideoBlocks := doc.Find(".tw-video")
	if twVideoBlocks.Length() > 0 {
		twVideoBlocks.Each(func(_ int, block *goquery.Selection) {
			blockThumb, _ := block.Find(".thumbnail img, .download-items__thumb img").Attr("src")

			var blockVideos []media.Item
			var blockPhotos []media.Item

			block.Find(".tw-button-dl").Each(func(_ int, s *goquery.Selection) {
				href, ok := s.Attr("href")
				if !ok || !strings.Contains(href, "dl.snapcdn.app") {
					return
				}
				text := strings.TrimSpace(s.Text())

				if strings.Contains(text, "MP4") {
					quality := ""
					if m := QualityRegex.FindStringSubmatch(text); len(m) > 1 {
						quality = m[1]
					}
					blockVideos = append(blockVideos, media.Item{
						Label:     fmt.Sprintf("Video %s", quality),
						Quality:   quality,
						URL:       href,
						Type:      media.Video,
						Thumbnail: blockThumb,
					})
				} else if strings.Contains(text, "图片") || strings.Contains(strings.ToLower(text), "photo") || strings.Contains(strings.ToLower(text), "image") {
					blockPhotos = append(blockPhotos, media.Item{
						Label:     "Photo",
						URL:       href,
						Type:      media.Image,
						Quality:   "original",
						Thumbnail: blockThumb,
					})
				}
			})

			if len(blockVideos) > 0 {
				// This block is a video. Select the highest quality variant and omit the preview thumbnail.
				resolution.SortItems(blockVideos)
				downloads = append(downloads, blockVideos[0])
			} else if len(blockPhotos) > 0 {
				// This block is a real photo without video.
				downloads = append(downloads, blockPhotos...)
			}
		})
	} else {
		// Fallback for HTML without .tw-video blocks
		var flatVideos []media.Item
		var flatPhotos []media.Item

		doc.Find(".tw-button-dl").Each(func(_ int, s *goquery.Selection) {
			href, ok := s.Attr("href")
			if !ok || !strings.Contains(href, "dl.snapcdn.app") {
				return
			}
			text := strings.TrimSpace(s.Text())

			if strings.Contains(text, "MP4") {
				quality := ""
				if m := QualityRegex.FindStringSubmatch(text); len(m) > 1 {
					quality = m[1]
				}
				flatVideos = append(flatVideos, media.Item{
					Label:   fmt.Sprintf("Video %s", quality),
					Quality: quality,
					URL:     href,
					Type:    media.Video,
				})
			} else if strings.Contains(text, "图片") || strings.Contains(strings.ToLower(text), "photo") || strings.Contains(strings.ToLower(text), "image") {
				flatPhotos = append(flatPhotos, media.Item{
					Label:   "Photo",
					URL:     href,
					Type:    media.Image,
					Quality: "original",
				})
			}
		})

		if len(flatVideos) > 0 {
			resolution.SortItems(flatVideos)
			downloads = append(downloads, flatVideos[0])
			for _, p := range flatPhotos {
				if !strings.Contains(p.URL, "video_thumb") {
					downloads = append(downloads, p)
				}
			}
		} else {
			downloads = append(downloads, flatPhotos...)
		}
	}

	doc.Find(".photo-list img").Each(func(_ int, s *goquery.Selection) {
		src, ok := s.Attr("src")
		if ok && src != "" {
			downloads = append(downloads, media.Item{
				Label:   "Photo",
				URL:     src,
				Type:    media.Image,
				Quality: "original",
			})
		}
	})

	hasVideo := false
	hasPhoto := false
	for _, d := range downloads {
		if d.Type == media.Video {
			hasVideo = true
		} else if d.Type == media.Image || d.Type == media.Slide {
			hasPhoto = true
		}
	}

	resultType := "photo"
	if hasVideo && hasPhoto {
		resultType = "multiple"
	} else if hasVideo {
		resultType = "video"
	}

	return &Result{
		Platform:  "twitter",
		Type:      resultType,
		TweetID:   tweetID,
		Title:     title,
		Duration:  duration,
		Thumbnail: thumbnail,
		Downloads: downloads,
	}
}
