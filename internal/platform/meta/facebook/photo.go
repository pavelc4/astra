package facebook

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
)

var (
	// Locates the album/carousel block FB server-renders into the page's
	// RelayPrefetchedStreamCache. count tells us how many photos the post has;
	// each subsequent viewer_image/image uri is one photo.
	albumBlockRe = regexp.MustCompile(`"all_subattachments":\{"count":(\d+)`)
	// viewer_image is the full-resolution variant (carries height/width); image
	// is the smaller thumbnail. Prefer viewer_image, fall back to image.
	// Photo nodes render viewer_image with dims first; a video node's poster
	// renders "viewer_image":{"uri":...} with NO dims, so this only ever matches
	// real photos — the discriminator that keeps video posters out of Photos.
	albumViewerRe = regexp.MustCompile(`"viewer_image":\{"height":(\d+),"width":(\d+),"uri":"(https:\\?/\\?/scontent[^"]+?)"`)
	albumThumbRe  = regexp.MustCompile(`"image":\{"uri":"(https:\\?/\\?/scontent[^"]+?)"`)
	photoBaseRe   = regexp.MustCompile(`/(\d{6,}_\d+_\d+)_`)
	// each subattachment node in all_subattachments.nodes[] begins with this key
	albumNodeRe = regexp.MustCompile(`"deduplication_key"`)
	// og:type is "video.*" for reels and single-video posts — the signal that a
	// non-album permalink is a video, not a photo set.
	ogTypeRe = regexp.MustCompile(`<meta[^>]*(?:property|content)="og:type"[^>]*content="([^"]+)"|<meta[^>]*content="([^"]+)"[^>]*(?:property|content)="og:type"`)

	// Single photo attachment in authenticated Comet HTML.
	// viewer_image has full dimensions (WxH); photo_image has the direct fbcdn URL.
	singlePhotoDimsRe = regexp.MustCompile(`"viewer_image":\{"height":(\d+),"width":(\d+)`)
	singlePhotoURLRe  = regexp.MustCompile(`"photo_image":\{"uri":"(https:\\?/\\?/scontent[^"]+?)"`)
)

// isVideoPage reports whether a crawled permalink is a video post (og:type
// video.*). Routes single videos to MP4 extraction instead of the photo scraper.
// Ignores group pages because Facebook sets og:type="video.other" indiscriminately
// on group pages even for photo and text posts.
func isVideoPage(htmlStr string) bool {
	if strings.Contains(htmlStr, `content="Grup Facebook"`) ||
		strings.Contains(htmlStr, `content="Facebook Groups"`) ||
		strings.Contains(htmlStr, `"application-name" content="Grup Facebook"`) ||
		strings.Contains(htmlStr, `"application-name" content="Facebook Groups"`) ||
		strings.Contains(htmlStr, `property="og:url" content="https://www.facebook.com/groups/`) ||
		strings.Contains(htmlStr, `property="og:url" content="https://web.facebook.com/groups/`) {
		return false
	}
	m := ogTypeRe.FindStringSubmatch(htmlStr)
	if len(m) < 3 {
		return false
	}
	val := m[1]
	if val == "" {
		val = m[2]
	}
	return strings.HasPrefix(val, "video")
}

func extractSinglePhoto(htmlStr string) *MediaItem {
	loc := strings.Index(htmlStr, "StoryAttachmentPhotoStyleRenderer")
	if loc == -1 {
		return nil
	}
	end := loc + 2000
	if end > len(htmlStr) {
		end = len(htmlStr)
	}
	chunk := htmlStr[loc:end]

	var quality string
	if m := singlePhotoDimsRe.FindStringSubmatch(chunk); len(m) > 2 {
		quality = m[2] + "x" + m[1]
	}
	if m := singlePhotoURLRe.FindStringSubmatch(chunk); len(m) > 1 {
		return &MediaItem{URL: cleanJSURL(m[1]), Quality: quality}
	}
	return nil
}

// extractAlbumMedia parses the multi-item carousel FB embeds in the
// authenticated permalink HTML (StoryAttachmentAlbumStyleRenderer), returning
// photos and videos separately. Each node in all_subattachments.nodes[] is a
// Photo or a Video; a Video's poster image must NOT be counted as a photo.
// Returns nil,nil if the page has no album block — callers fall back to og:image.
//
// ponytail: scrapes the server-rendered Relay payload, not the GraphQL API —
// FB killed arbitrary q= queries (doc_id-only now), but still SSRs the result
// into the page. If FB stops embedding it, switch to a persisted doc_id call.
func extractAlbumMedia(htmlStr string) (photos, videos []MediaItem) {
	loc := albumBlockRe.FindStringSubmatchIndex(htmlStr)
	if loc == nil {
		return nil, nil
	}
	count := 0
	for _, c := range htmlStr[loc[2]:loc[3]] {
		count = count*10 + int(c-'0')
	}
	if count < 1 {
		return nil, nil
	}

	// Split the node array by its per-node "deduplication_key" marker and cap at
	// count so we don't bleed into a neighbouring post's attachments further down.
	tail := htmlStr[loc[1]:]
	starts := albumNodeRe.FindAllStringIndex(tail, count+1)
	if len(starts) == 0 {
		return nil, nil
	}

	seen := make(map[string]bool)
	for i := 0; i < count && i < len(starts); i++ {
		end := len(tail)
		if i+1 < len(starts) {
			end = starts[i+1][0]
		}
		node := tail[starts[i][0]:end]

		if strings.Contains(node, `"__typename":"Video"`) {
			// ponytail: take the best progressive MP4 (HD first) FB SSRs into the
			// node; browser_native_{hd,sd}_url is the fallback for DASH-only videos
			// that carry no progressive_urls. Skips a video only if neither is
			// present. Add dash_manifests base_url parsing if such videos show up.
			v := extractProgressiveVideos(node)
			if len(v) == 0 {
				v = extractBrowserNativeVideos(node)
			}
			if len(v) > 0 {
				item := v[0]
				if poster := firstAlbumImage(node); poster != "" {
					item.Thumbnail = &poster
				}
				videos = append(videos, item)
			}
			continue
		}

		// Photo node: prefer full-res viewer_image (WxH), fall back to image.
		if m := albumViewerRe.FindStringSubmatch(node); len(m) > 3 {
			clean := cleanJSURL(m[3])
			if base := photoBaseName(clean); base != "" && !seen[base] {
				seen[base] = true
				photos = append(photos, MediaItem{URL: clean, Quality: m[2] + "x" + m[1]}) // WxH
				continue
			}
		}
		if u := firstAlbumImage(node); u != "" {
			if base := photoBaseName(u); base != "" && !seen[base] {
				seen[base] = true
				photos = append(photos, MediaItem{URL: u})
			}
		}
	}
	return photos, videos
}

// firstAlbumImage returns the first fbcdn image uri in a node — a photo's image
// or a video's poster.
func firstAlbumImage(node string) string {
	if m := albumThumbRe.FindStringSubmatch(node); len(m) > 1 {
		return cleanJSURL(m[1])
	}
	return ""
}

// photoBaseName returns the fbcdn filename (123_456_789_n) so the same photo at
// thumb and full-res resolution dedupes to one entry.
func photoBaseName(u string) string {
	m := photoBaseRe.FindStringSubmatch(u)
	if m == nil {
		return ""
	}
	return m[1]
}

func fetchPhotosViaCrawler(ctx context.Context, targetURL, ck string, wantVideo bool) (*MediaInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if ck != "" {
		// Full browser headers get FB to server-render the album (bbox) instead
		// of a bare og:image, and avoid the 400 "something went wrong" page.
		setBrowserHeaders(req)
		req.Header.Set("Cookie", ck)
	} else {
		req.Header.Set("User-Agent", "facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_voiced.php)")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	htmlStr := string(body)
	info := &MediaInfo{}

	// Extract caption
	titleRe := regexp.MustCompile(`<meta[^>]*(?:property|content)="og:title"[^>]*content="([^"]+)"|<meta[^>]*content="([^"]+)"[^>]*(?:property|content)="og:title"`)
	if t := titleRe.FindStringSubmatch(htmlStr); len(t) > 0 {
		val := t[1]
		if val == "" {
			val = t[2]
		}
		info.Caption = html.UnescapeString(val)
	}

	// Multi-item carousel: prefer the server-rendered album block (all N items,
	// photos and videos) over og:image (which only ever carries the single cover).
	if photos, videos := extractAlbumMedia(htmlStr); len(photos)+len(videos) > 0 {
		info.Photos = photos
		info.Videos = videos
		if len(photos) > 0 {
			info.Thumbnail = &info.Photos[0].URL
		} else if videos[0].Thumbnail != nil {
			info.Thumbnail = videos[0].Thumbnail
		}
		if info.Caption == "" {
			info.Caption = extractVideoCaption(htmlStr)
		}
		return info, nil
	}

	// Single photo from authenticated Comet SSR:
	if !wantVideo && !isVideoPage(htmlStr) {
		if item := extractSinglePhoto(htmlStr); item != nil {
			info.Photos = []MediaItem{*item}
			info.Thumbnail = &item.URL
			if info.Caption == "" {
				info.Caption = extractVideoCaption(htmlStr)
			}
			return info, nil
		}
	}

	// Single video posts (a /share/v/ that resolves to a group permalink, a lone
	// reel) render NO album block and ship the MP4 as progressive_url /
	// browser_native_* in the page. Without this, the og:image + comet fallback
	// below scrapes the poster and every feed image instead of the video.
	//
	// Gated on wantVideo (caller knows this came from a /share/v//r/ link) or
	// og:type=video on the lighter public page. NOT on progressive presence
	// alone: the authenticated Comet page carries progressive_urls for a dozen
	// suggested videos too, so an ungated grab would hijack real photo posts.
	if wantVideo || isVideoPage(htmlStr) {
		vids := extractProgressiveVideos(htmlStr)
		if len(vids) == 0 {
			vids = extractBrowserNativeVideos(htmlStr)
		}
		if len(vids) > 0 {
			thRe := regexp.MustCompile(`"preferred_thumbnail":\{"image":\{"uri":"([^"]+)"`)
			if thMatches := thRe.FindAllStringSubmatchIndex(htmlStr, -1); len(thMatches) > 0 {
				for _, th := range thMatches {
					if len(th) >= 4 && !isAdContext(htmlStr, th[2]) {
						thumb := cleanJSURL(htmlStr[th[2]:th[3]])
						info.Thumbnail = &thumb
						for i := range vids {
							vids[i].Thumbnail = &thumb
						}
						break
					}
				}
			}
			info.Videos = vids
			info.Photos = nil
			if info.Caption == "" {
				info.Caption = extractVideoCaption(htmlStr)
			}
			return info, nil
		}
	}

	// Extract photos
	ogImageRe := regexp.MustCompile(`<meta[^>]*(?:property|content)="og:image"[^>]*content="([^"]+)"|<meta[^>]*content="([^"]+)"[^>]*(?:property|content)="og:image"`)
	ogMatches := ogImageRe.FindAllStringSubmatch(htmlStr, -1)

	seen := make(map[string]bool)
	for _, m := range ogMatches {
		val := m[1]
		if val == "" {
			val = m[2]
		}
		clean := strings.ReplaceAll(val, `\/`, `/`)
		clean = strings.ReplaceAll(clean, `&amp;`, `&`)
		if strings.Contains(clean, "static.xx") || strings.Contains(clean, "rsrc.php") {
			continue
		}
		if seen[clean] {
			continue
		}
		seen[clean] = true
		info.Photos = append(info.Photos, MediaItem{URL: clean})
	}

	// Fallback photo extraction for dynamically loaded/Comet page HTML (e.g. private groups/posts).
	// Skipped for video pages: this scrape grabs every fbcdn image on the page —
	// profile pics, cover photos, feed thumbnails — and for a video-as-post that
	// means dozens of junk images, some at sizes the CDN 403s. A video with no
	// extractable URL returns nothing here and falls through to a clean error.
	if len(info.Photos) == 0 && !wantVideo && !isVideoPage(htmlStr) {
		// Post photos typically reside under /t39.xxxx or /v/t39.xxxx and have _n.jpg/_n.png/_n.webp in their filenames
		cometImageRe := regexp.MustCompile(`https?:\\?/\\?/[^\s"'>]+?fbcdn\.net[^\s"'>]+?(?:_n\.[a-zA-Z0-9]+|\/t39\.[^\s"'>]+?\.[a-zA-Z0-9]+)[^\s"'>]*`)
		cometMatches := cometImageRe.FindAllString(htmlStr, -1)

		for _, m := range cometMatches {
			clean := cleanJSURL(m)
			// Skip icons, profile pictures and small thumbnails
			if strings.Contains(clean, "static.xx") || strings.Contains(clean, "rsrc.php") || strings.Contains(clean, "/cp0/") || strings.Contains(clean, "/s40x40/") || strings.Contains(clean, "/s100x100/") {
				continue
			}
			if seen[clean] {
				continue
			}
			seen[clean] = true
			info.Photos = append(info.Photos, MediaItem{URL: clean})
		}
	}

	if len(info.Photos) > 0 {
		info.Thumbnail = &info.Photos[0].URL
	}
	if info.Caption == "" {
		info.Caption = extractVideoCaption(htmlStr)
	}

	return info, nil
}
