package facebook

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var graphQLURL = "https://www.facebook.com/api/graphql/"

var (
	videoIDPatterns = []*regexp.Regexp{
		regexp.MustCompile(`/videos/(?:[^/]+/)?(\d+)/?`),
		regexp.MustCompile(`/reel/(\d+)`),
		regexp.MustCompile(`/watch/?\?.*?v=(\d+)`),
		regexp.MustCompile(`/story\.php\?.*?story_fbid=([^&]+)`),
		regexp.MustCompile(`/permalink\.php\?.*?story_fbid=([^&]+)`),
		regexp.MustCompile(`/posts/([^/?#]+)`),
		regexp.MustCompile(`/photo(?:\.php)?\?.*?fbid=([^&]+)`),
		regexp.MustCompile(`/photos/([^/?#]+)`),
	}

	progressiveURLRe = regexp.MustCompile(`"progressive_url":"(https:\\?/\\?/[^"]+?)"[^}]*?"quality":"(\w+)"`)

	// browser_native_{hd,sd}_url are the direct-download MP4s FB server-renders
	// for reels and many feed/group videos that ship NO progressive_urls (they
	// deliver via DASH: dash_manifest + base_url). Same HD-preferred shape.
	browserNativeHDRe = regexp.MustCompile(`"browser_native_hd_url":"(https:\\?/\\?/[^"]+?)"`)
	browserNativeSDRe = regexp.MustCompile(`"browser_native_sd_url":"(https:\\?/\\?/[^"]+?)"`)
)

func extractVideoID(ctx context.Context, rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	path := u.Path

	if strings.HasPrefix(u.Host, "fb.watch") || strings.HasPrefix(u.Host, "www.fb.watch") || strings.Contains(path, "/fb.watch/") {
		return resolveShortURL(ctx, rawURL, u.Host)
	}

	if strings.Contains(path, "/share/") {
		return resolveShortURL(ctx, rawURL, u.Host)
	}

	target := path
	if u.RawQuery != "" {
		target = path + "?" + u.RawQuery
	}

	for _, pat := range videoIDPatterns {
		if m := pat.FindStringSubmatch(target); len(m) > 1 {
			return m[1], nil
		}
	}

	return "", fmt.Errorf("could not extract video ID from URL: %s", rawURL)
}

func resolveShortURL(ctx context.Context, rawURL, host string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	finalURL := resp.Request.URL.String()
	if finalURL != rawURL {
		return extractVideoID(ctx, finalURL)
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	html := string(body)

	canonical := regexp.MustCompile(`<link[^>]+rel="canonical"[^>]+href="([^"]+)"`).FindStringSubmatch(html)
	if len(canonical) > 1 {
		return extractVideoID(ctx, canonical[1])
	}

	ogURL := regexp.MustCompile(`<meta[^>]+property="og:url"[^>]+content="([^"]+)"`).FindStringSubmatch(html)
	if len(ogURL) > 1 {
		return extractVideoID(ctx, ogURL[1])
	}

	return "", fmt.Errorf("could not resolve short URL: %s", rawURL)
}

func fetchPageAndDTSG(ctx context.Context, videoID, ck string) (string, string, error) {
	pageURL := fmt.Sprintf("https://www.facebook.com/watch/?v=%s", videoID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("create page request: %w", err)
	}
	req.Header.Set("Cookie", ck)
	setBrowserHeaders(req)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("page request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return "", "", fmt.Errorf("read page: %w", err)
	}

	html := string(body)

	dtsgRe := regexp.MustCompile(`"DTSGInitialData"\s*,\s*\[\]\s*,\s*{\s*"token"\s*:\s*"([^"]+)"`)
	dtsg := ""
	if m := dtsgRe.FindStringSubmatch(html); len(m) > 1 && m[1] != "" {
		dtsg = m[1]
	}

	return html, dtsg, nil
}

// extractVideoCaption pulls the reel/video caption. The watch page <title> is
// just "Video"; the real text lives in the story's message.text (JSON-escaped).
//
// ponytail: takes the first message.text on the page (the primary video's
// creation story precedes comments in the SSR). If comments start leaking in,
// scope it to the videoDeliveryResponseFragment's owning story.
func extractVideoCaption(htmlStr string) string {
	if m := regexp.MustCompile(`"message":\{"text":"((?:[^"\\]|\\.)*)"`).FindStringSubmatch(htmlStr); len(m) > 1 {
		var s string
		if err := json.Unmarshal([]byte(`"`+m[1]+`"`), &s); err == nil && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	if t := regexp.MustCompile(`<title[^>]*>([^<]+)</title>`).FindStringSubmatch(htmlStr); len(t) > 1 {
		return html.UnescapeString(strings.TrimSpace(t[1]))
	}
	return ""
}

// isAdContext reports whether the substring around matchStart appears to be
// part of a sponsored/ad placement in Facebook Comet HTML.
func isAdContext(htmlStr string, matchStart int) bool {
	start := matchStart - 2000
	if start < 0 {
		start = 0
	}
	end := matchStart + 1000
	if end > len(htmlStr) {
		end = len(htmlStr)
	}
	window := htmlStr[start:end]
	return strings.Contains(window, "VIDEO_DR_STYLE") ||
		strings.Contains(window, "StoryAttachmentLinkOpenCallToActionRenderer") ||
		strings.Contains(window, `"link_style":"VIDEO_DR_STYLE"`) ||
		strings.Contains(window, `"is_sponsored":true`)
}

// extractProgressiveVideos pulls the direct-download MP4 URLs FB server-renders
// into the watch page (progressive_urls with HD/SD quality labels). Returns nil
// if none present — callers fall back to the legacy embed/GraphQL paths.
//
// ponytail: modern FB videos ship via DASH; plugins/video.php (hd_src/sd_src)
// now returns "unavailable" for many of them. The progressive_urls in the SSR
// payload are the reliable source. If FB drops them, the fallback is parsing the
// DASH base_url representations.
func extractProgressiveVideos(htmlStr string) []MediaItem {
	matches := progressiveURLRe.FindAllStringSubmatchIndex(htmlStr, -1)
	if matches == nil {
		return nil
	}
	seen := make(map[string]bool)
	var hd, sd []MediaItem
	for _, m := range matches {
		if len(m) < 6 {
			continue
		}
		urlStart, urlEnd := m[2], m[3]
		qStart, qEnd := m[4], m[5]
		if isAdContext(htmlStr, urlStart) {
			continue
		}
		rawURL := htmlStr[urlStart:urlEnd]
		clean := cleanJSURL(rawURL)
		q := strings.ToLower(htmlStr[qStart:qEnd])
		if seen[q] {
			continue
		}
		seen[q] = true
		item := MediaItem{Quality: q, URL: clean}
		if q == "hd" {
			hd = append(hd, item)
		} else {
			sd = append(sd, item)
		}
	}
	// HD first so the best quality is the primary source.
	return append(hd, sd...)
}

// extractBrowserNativeVideos pulls the direct MP4s FB exposes as
// browser_native_hd_url / browser_native_sd_url. This is the fallback when a
// video ships no progressive_urls (DASH-only delivery) — the case that made
// reels fail intermittently and group videos fall back to a poster image.
// Takes the FIRST non-ad match of each.
func extractBrowserNativeVideos(htmlStr string) []MediaItem {
	var out []MediaItem
	if hdMatches := browserNativeHDRe.FindAllStringSubmatchIndex(htmlStr, -1); len(hdMatches) > 0 {
		for _, m := range hdMatches {
			if len(m) >= 4 && !isAdContext(htmlStr, m[2]) {
				out = append(out, MediaItem{Quality: "hd", URL: cleanJSURL(htmlStr[m[2]:m[3]])})
				break
			}
		}
	}
	if sdMatches := browserNativeSDRe.FindAllStringSubmatchIndex(htmlStr, -1); len(sdMatches) > 0 {
		for _, m := range sdMatches {
			if len(m) >= 4 && !isAdContext(htmlStr, m[2]) {
				out = append(out, MediaItem{Quality: "sd", URL: cleanJSURL(htmlStr[m[2]:m[3]])})
				break
			}
		}
	}
	return out
}

func queryGraphQL(ctx context.Context, videoID, dtsg, ck string) (*MediaInfo, error) {
	query := `query($id: ID!) { node(id: $id) { __typename ... on Video { playable_url playable_url_quality_hd browser_native_hd_url browser_native_sd_url title preferred_thumbnail { image { uri } } playable_duration_in_ms } } }`

	variables := map[string]string{"id": videoID}
	varBytes, _ := json.Marshal(variables)

	form := url.Values{
		"q":         {query},
		"variables": {string(varBytes)},
	}

	if dtsg != "" {
		form.Set("fb_dtsg", dtsg)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, graphQLURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create graphql request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Cookie", ck)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("graphql request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read graphql response: %w", err)
	}

	var gql graphQLResponse
	if err := json.Unmarshal(body, &gql); err != nil {
		return nil, fmt.Errorf("parse graphql response: %w", err)
	}

	if len(gql.Errors) > 0 {
		return nil, fmt.Errorf("graphql error: %s", gql.Errors[0].Message)
	}

	if gql.Data == nil || gql.Data.Node == nil {
		return nil, fmt.Errorf("graphql: video not found")
	}

	node := gql.Data.Node
	if node.Typename != "Video" {
		return nil, fmt.Errorf("graphql: expected Video, got %s", node.Typename)
	}

	videoURL := firstNonEmpty(node.PlayableHD, node.PlayableURL, node.BrowserHD, node.BrowserSD)
	if videoURL == "" {
		return nil, fmt.Errorf("graphql: no playable URL found")
	}

	info := &MediaInfo{
		Caption: firstNonEmpty(node.Title, node.Description),
	}

	if node.DurationMs > 0 {
		info.Duration = fmt.Sprintf("%d", node.DurationMs/1000)
	}

	if node.Thumbnail != nil && node.Thumbnail.Image != nil && node.Thumbnail.Image.URI != "" {
		info.Thumbnail = &node.Thumbnail.Image.URI
	}

	info.Videos = []MediaItem{
		{Quality: "hd", URL: videoURL, Thumbnail: info.Thumbnail},
	}

	return info, nil
}

func extractFromPage(html string, videoID string) (*MediaInfo, error) {
	// Attempt to locate playable URLs embedded in RelayPrefetchedData.
	// This is the same approach yt-dlp uses as its primary extraction path.
	hdRe := regexp.MustCompile(`"playable_url_quality_hd":"(https?:\\?/\\?/[^"]+)"`)
	sdRe := regexp.MustCompile(`"playable_url":"(https?:\\?/\\?/[^"]+)"`)

	var videoURL string
	var quality string

	if m := hdRe.FindStringSubmatch(html); len(m) > 1 {
		videoURL = m[1]
		quality = "hd"
	} else if m := sdRe.FindStringSubmatch(html); len(m) > 1 {
		videoURL = m[1]
		quality = "sd"
	}

	if videoURL == "" {
		return nil, fmt.Errorf("no playable URL found in page data")
	}

	info := &MediaInfo{}
	clean := strings.ReplaceAll(videoURL, `\/`, `/`)
	info.Videos = []MediaItem{
		{Quality: quality, URL: clean},
	}

	titleRe := regexp.MustCompile(`"name":"([^"]+)"`)
	if t := titleRe.FindStringSubmatch(html); len(t) > 1 {
		info.Caption = t[1]
	}

	thumbRe := regexp.MustCompile(`"preferred_thumbnail":\{"image":\{"uri":"([^"]+)"\}`)
	if t := thumbRe.FindStringSubmatch(html); len(t) > 1 {
		thumb := strings.ReplaceAll(t[1], `\/`, `/`)
		info.Thumbnail = &thumb
	}

	return info, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return strings.ReplaceAll(v, `\/`, `/`)
		}
	}
	return ""
}

func fetchVideoViaEmbed(ctx context.Context, targetURL, ck string) (*MediaInfo, error) {
	embedURL := fmt.Sprintf("https://www.facebook.com/plugins/video.php?href=%s", url.QueryEscape(targetURL))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, embedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	if ck != "" {
		req.Header.Set("Cookie", ck)
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

	// Extract sources
	hdRe := regexp.MustCompile(`"hd_src":"([^"]+)"`)
	sdRe := regexp.MustCompile(`"sd_src":"([^"]+)"`)

	var hdURL, sdURL string
	if m := hdRe.FindStringSubmatch(htmlStr); len(m) > 1 {
		hdURL = cleanJSURL(m[1])
	}
	if m := sdRe.FindStringSubmatch(htmlStr); len(m) > 1 {
		sdURL = cleanJSURL(m[1])
	}

	if hdURL == "" && sdURL == "" {
		return nil, fmt.Errorf("no video sources found in embed player")
	}

	info := &MediaInfo{}
	if hdURL != "" {
		info.Videos = append(info.Videos, MediaItem{Quality: "hd", URL: hdURL})
	} else if sdURL != "" {
		info.Videos = append(info.Videos, MediaItem{Quality: "sd", URL: sdURL})
	}

	// Try fetching title/caption from watch/reel page as crawler. Video already
	// resolved above via the embed player, so this only needs caption/thumbnail —
	// wantVideo=false keeps it out of the crawler's video-extraction branch.
	crawlerInfo, err := fetchPhotosViaCrawler(ctx, targetURL, "", false)
	if err != nil && ck != "" {
		crawlerInfo, err = fetchPhotosViaCrawler(ctx, targetURL, ck, false)
	}
	if err == nil {
		info.Caption = crawlerInfo.Caption
		info.Thumbnail = crawlerInfo.Thumbnail
	} else {
		// Fallback thumbnail/caption extraction from embed page
		titleRe := regexp.MustCompile(`<meta[^>]*(?:property|content)="og:title"[^>]*content="([^"]+)"|<meta[^>]*content="([^"]+)"[^>]*(?:property|content)="og:title"`)
		if t := titleRe.FindStringSubmatch(htmlStr); len(t) > 0 {
			val := t[1]
			if val == "" {
				val = t[2]
			}
			info.Caption = val
		}

		jpgRe := regexp.MustCompile(`"https?:\\?/\\?/[^\s"'>]+?\.jpg[^\s"'>]*"`)
		if m := jpgRe.FindString(htmlStr); m != "" {
			clean := cleanJSURL(strings.Trim(m, `"`))
			info.Thumbnail = &clean
		}
	}

	return info, nil
}
