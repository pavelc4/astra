package pixiv

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/httpclient"
	"github.com/pavelc4/astra/internal/media"
)

// pixivHTTPClient short-circuits redirects so we see the literal CDN/API status
// (a 403 referer rejection comes BEFORE any redirect) and can report it.
var pixivHTTPClient = &http.Client{
	Timeout:   15 * time.Second,
	Transport: httpclient.Client.Transport,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// pageURL returns the Nth page's original URL. Pixiv names pages _p0, _p1, ...
// from a single template where the first page's URL already ends in p0.
func pageURL(page int, original string) string {
	if page == 0 {
		return original
	}
	return strings.Replace(original, "p0", "p"+strconv.Itoa(page), 1)
}

func FetchData(ctx context.Context, targetURL string) (*media.Media, error) {
	m := reArtworkID.FindStringSubmatch(targetURL)
	if m == nil {
		return nil, errors.NewInvalidURL("not a valid Pixiv artwork URL (expected /artworks/{id})")
	}
	id := m[1]

	body, err := get(ctx, fmt.Sprintf("/ajax/illust/%s?time=%d", id, time.Now().UnixMilli()))
	if err != nil {
		return nil, err
	}

	var data illustResp
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, errors.NewUpstream(err.Error())
	}
	if data.Error || data.Body == nil {
		return nil, errors.NewUpstream(data.Message)
	}

	b := data.Body
	title := b.Title
	if b.UserName != "" {
		title = b.Title + " - " + b.UserName
	}

	var items []media.Item
	if b.IllustType == 2 { // ugoira: animated work
		meta, err := FetchUgoiraMeta(ctx, id)
		if err == nil && meta != nil && meta.OriginalSrc != "" {
			items = append(items, media.Item{
				Label:   "Ugoira Original (Zip)",
				URL:     meta.OriginalSrc,
				Type:    media.Video,
				Quality: "original",
			})
			if meta.Src != "" && meta.Src != meta.OriginalSrc {
				items = append(items, media.Item{
					Label:   "Ugoira 600x600 (Zip)",
					URL:     meta.Src,
					Type:    media.Video,
					Quality: "600x600",
				})
			}
		} else {
			items = append(items, media.Item{
				Label:   "Ugoira",
				URL:     b.URLs.Original,
				Type:    media.Video,
				Quality: "ugoira",
			})
		}
	} else {
		pages := b.PageCount
		if pages < 1 {
			pages = 1
		}
		for i := 0; i < pages; i++ {
			items = append(items, media.Item{
				Label:   "Page " + strconv.Itoa(i+1),
				URL:     pageURL(i, b.URLs.Original),
				Type:    media.Image,
				Quality: "original",
			})
		}
	}

	if len(items) == 0 {
		return nil, errors.NewUpstream(fmt.Sprintf("no media found for artwork %s", id))
	}

	return media.Downloads("pixiv", title, b.URLs.Regular, items), nil
}

// FetchUserProfile returns the user's artwork list. Accepts any /users/{id} URL.
func FetchUserProfile(ctx context.Context, targetURL string) ([]Artwork, error) {
	return fetchUserWorks(ctx, targetURL, "/illusts/tag")
}

// FetchUserIllustrations returns the user's illustrations. Accepts
// /users/{id}/illustrations (and /users/{id}).
func FetchUserIllustrations(ctx context.Context, targetURL string) ([]Artwork, error) {
	return fetchUserWorks(ctx, targetURL, "/illusts/tag")
}

// FetchUserBookmarks returns a user's public bookmark list. Accepts
// /users/{id}/bookmarks/artworks. Requires a (logged-in) cookie for remote users.
func FetchUserBookmarks(ctx context.Context, targetURL string) ([]Artwork, error) {
	m := reUserID.FindStringSubmatch(targetURL)
	if m == nil {
		return nil, errors.NewInvalidURL("not a valid Pixiv user URL (expected /users/{id})")
	}
	return fetchWorks(ctx, m[1], "/illusts/bookmarks", "rest=show&offset=0&limit=100&tag=")
}

// fetchUserWorks reads userID from the URL then lists that user's illustrations.
func fetchUserWorks(ctx context.Context, targetURL, suffix string) ([]Artwork, error) {
	m := reUserID.FindStringSubmatch(targetURL)
	if m == nil {
		return nil, errors.NewInvalidURL("not a valid Pixiv user URL (expected /users/{id})")
	}
	return fetchWorks(ctx, m[1], suffix, "tag=&offset=0&limit=100")
}

func fetchWorks(ctx context.Context, userID, suffix, query string) ([]Artwork, error) {
	body, err := get(ctx, fmt.Sprintf("/ajax/user/%s%s?%s", userID, suffix, query))
	if err != nil {
		return nil, err
	}
	var data worksResp
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, errors.NewUpstream(err.Error())
	}
	if data.Error {
		return nil, errors.NewUpstream(data.Message)
	}

	works := make([]Artwork, 0, len(data.Body.Works))
	for _, w := range data.Body.Works {
		works = append(works, Artwork{
			ID:      w.ID,
			Title:   w.Title,
			Thumb:   w.URLs.Regular,
			PageURL: "https://www.pixiv.net/artworks/" + w.ID,
		})
	}
	return works, nil
}

// FetchUgoiraMeta retrieves animation metadata and frames from /ajax/illust/{id}/ugoira_meta.
func FetchUgoiraMeta(ctx context.Context, illustID string) (*UgoiraMeta, error) {
	body, err := get(ctx, fmt.Sprintf("/ajax/illust/%s/ugoira_meta", illustID))
	if err != nil {
		return nil, err
	}
	var data ugoiraResp
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, errors.NewUpstream(err.Error())
	}
	if data.Error || data.Body == nil {
		return nil, errors.NewUpstream(data.Message)
	}
	return data.Body, nil
}

// FetchUgoiraDirect extracts the animation metadata directly from an artwork URL.
func FetchUgoiraDirect(ctx context.Context, targetURL string) (*UgoiraMeta, error) {
	m := reArtworkID.FindStringSubmatch(targetURL)
	if m == nil {
		return nil, errors.NewInvalidURL("not a valid Pixiv artwork URL (expected /artworks/{id})")
	}
	return FetchUgoiraMeta(ctx, m[1])
}

// FetchRankings gets daily, weekly, or monthly artwork rankings from /ranking.php?mode=...&format=json.
func FetchRankings(ctx context.Context, mode string, page int) ([]RankingItem, error) {
	if mode == "" || !validRankingModes[mode] {
		mode = "daily"
	}
	if page < 1 {
		page = 1
	}

	body, err := get(ctx, fmt.Sprintf("/ranking.php?mode=%s&p=%d&format=json", mode, page))
	if err != nil {
		return nil, err
	}

	var data rankingResp
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, errors.NewUpstream(err.Error())
	}

	items := make([]RankingItem, len(data.Contents))
	for i, c := range data.Contents {
		pc, _ := strconv.Atoi(c.PageCount.String())
		items[i] = RankingItem{
			Rank:        c.Rank,
			IllustID:    c.IllustID,
			Title:       c.Title,
			UserID:      c.UserID,
			UserName:    c.UserName,
			URL:         c.URL,
			PageURL:     fmt.Sprintf("https://www.pixiv.net/artworks/%d", c.IllustID),
			PageCount:   pc,
			ViewCount:   c.ViewCount,
			RatingCount: c.RatingCount,
			Tags:        c.Tags,
		}
	}

	return items, nil
}

// FetchUserAllWorks retrieves all illust, manga, and novel IDs for a creator via /ajax/user/{id}/profile/all.
func FetchUserAllWorks(ctx context.Context, targetURL string) (*UserAllWorks, error) {
	m := reUserID.FindStringSubmatch(targetURL)
	if m == nil {
		return nil, errors.NewInvalidURL("not a valid Pixiv user URL (expected /users/{id})")
	}
	userID := m[1]

	body, err := get(ctx, fmt.Sprintf("/ajax/user/%s/profile/all", userID))
	if err != nil {
		return nil, err
	}

	var data userProfileAllResp
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, errors.NewUpstream(err.Error())
	}
	if data.Error {
		return nil, errors.NewUpstream(data.Message)
	}

	return &UserAllWorks{
		Illusts: extractMapKeys(data.Body.Illusts),
		Manga:   extractMapKeys(data.Body.Manga),
		Novels:  extractMapKeys(data.Body.Novels),
	}, nil
}

func extractMapKeys(v any) []string {
	m, ok := v.(map[string]any)
	if !ok {
		return []string{}
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// FetchNovel retrieves full novel text content and metadata from /ajax/novel/{id}.
func FetchNovel(ctx context.Context, targetURL string) (*Novel, error) {
	m := reNovelID.FindStringSubmatch(targetURL)
	if m == nil {
		return nil, errors.NewInvalidURL("not a valid Pixiv novel URL (expected /novel/show.php?id={id})")
	}
	id := m[1]

	body, err := get(ctx, fmt.Sprintf("/ajax/novel/%s?time=%d", id, time.Now().UnixMilli()))
	if err != nil {
		return nil, err
	}

	var data novelResp
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, errors.NewUpstream(err.Error())
	}
	if data.Error {
		return nil, errors.NewUpstream(data.Message)
	}

	b := data.Body
	tags := make([]string, len(b.Tags.Tags))
	for i, t := range b.Tags.Tags {
		tags[i] = t.Tag
	}

	return &Novel{
		ID:             b.ID,
		Title:          b.Title,
		Description:    b.Description,
		Content:        b.Content,
		CoverURL:       b.CoverURL,
		UserID:         b.UserID,
		UserName:       b.UserName,
		WordCount:      b.WordCount,
		CharacterCount: b.CharacterCount,
		ReadingTime:    b.ReadingTime,
		Tags:           tags,
		PageURL:        fmt.Sprintf("https://www.pixiv.net/novel/show.php?id=%s", b.ID),
	}, nil
}

// SearchArtworks searches artworks by tag or keyword from /ajax/search/artworks/{word}.
func SearchArtworks(ctx context.Context, word string, page int, mode string) ([]SearchItem, error) {
	if word == "" {
		return nil, errors.NewValidation("search keyword 'word' is required")
	}
	if page < 1 {
		page = 1
	}
	if mode != "safe" && mode != "r18" {
		mode = "all"
	}

	escaped := url.PathEscape(word)
	path := fmt.Sprintf("/ajax/search/artworks/%s?word=%s&order=date_d&mode=%s&p=%d", escaped, url.QueryEscape(word), mode, page)

	body, err := get(ctx, path)
	if err != nil {
		return nil, err
	}

	var data searchResp
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, errors.NewUpstream(err.Error())
	}
	if data.Error {
		return nil, errors.NewUpstream(data.Message)
	}

	rawItems := data.Body.IllustManga.Data
	items := make([]SearchItem, len(rawItems))
	for i, it := range rawItems {
		items[i] = SearchItem{
			ID:        it.ID,
			Title:     it.Title,
			ThumbURL:  it.URL,
			PageURL:   fmt.Sprintf("https://www.pixiv.net/artworks/%s", it.ID),
			UserID:    it.UserID,
			UserName:  it.UserName,
			PageCount: it.PageCount,
			Tags:      it.Tags,
			AIType:    it.AIType,
			XRestrict: it.XRestrict,
		}
	}

	return items, nil
}

// FetchUserDetail retrieves comprehensive creator profile details from /ajax/user/{id}?full=1.
func FetchUserDetail(ctx context.Context, targetURL string) (*UserDetail, error) {
	m := reUserID.FindStringSubmatch(targetURL)
	if m == nil {
		return nil, errors.NewInvalidURL("not a valid Pixiv user URL (expected /users/{id})")
	}
	userID := m[1]

	body, err := get(ctx, fmt.Sprintf("/ajax/user/%s?full=1", userID))
	if err != nil {
		return nil, err
	}

	var data userDetailResp
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, errors.NewUpstream(err.Error())
	}
	if data.Error {
		return nil, errors.NewUpstream(data.Message)
	}

	b := data.Body
	social := make(map[string]string)
	if m, ok := b.Social.(map[string]any); ok {
		for platform, rawLink := range m {
			if linkMap, ok := rawLink.(map[string]any); ok {
				if u, ok := linkMap["url"].(string); ok && u != "" {
					social[platform] = u
				}
			}
		}
	}

	regionStr := ""
	if b.Region != nil && b.Region.Name != nil {
		regionStr = *b.Region.Name
	}

	return &UserDetail{
		UserID:    b.UserID,
		Name:      b.Name,
		Avatar:    b.Image,
		AvatarBig: b.ImageBig,
		Comment:   b.Comment,
		Webpage:   b.Webpage,
		Social:    social,
		Following: b.Following,
		Premium:   b.Premium,
		Region:    regionStr,
	}, nil
}

// get performs an authenticated ajax GET and returns the raw response body.
func get(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
	if err != nil {
		return nil, errors.NewUpstream(err.Error())
	}

	req.Header.Set("Referer", referer)
	req.Header.Set("Accept", "application/json")
	for k, v := range httpclient.DefaultHeaders {
		req.Header[k] = v
	}
	if c := GetCookies(); c != "" {
		req.Header.Set("Cookie", c)
	}

	resp, err := pixivHTTPClient.Do(req)
	if err != nil {
		return nil, errors.NewUpstream(err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewUpstream(err.Error())
	}

	if resp.StatusCode == 403 || resp.StatusCode == 400 {
		return nil, errors.NewUpstream(fmt.Sprintf("pixiv %d: login or invalid/expired cookie required", resp.StatusCode))
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.NewUpstream(fmt.Sprintf("pixiv returned %d", resp.StatusCode))
	}
	return body, nil
}

// StreamImage proxies an i.pximg.net image to the client, adding the Referer
// header Pixiv's hotlink protection requires. Returns the resolved content-type
// so the handler can set it on the response.
func StreamImage(ctx context.Context, imageURL string, out io.Writer) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return "", errors.NewUpstream(err.Error())
	}

	req.Header.Set("Referer", referer)
	req.Header.Set("Accept", "image/avif,image/webp,image/png,image/jpeg,*/*;q=0.8")
	for k, v := range httpclient.DefaultHeaders {
		req.Header[k] = v
	}
	if c := GetCookies(); c != "" {
		req.Header.Set("Cookie", c)
	}

	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return "", errors.NewUpstream(err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 || resp.StatusCode == 400 {
		return "", errors.NewUpstream(fmt.Sprintf("pixiv image %d: referer rejected or cookie required", resp.StatusCode))
	}
	if resp.StatusCode != http.StatusOK {
		return "", errors.NewUpstream(fmt.Sprintf("pixiv image returned %d", resp.StatusCode))
	}

	ctype := resp.Header.Get("Content-Type")
	if ctype == "" {
		ctype = "application/octet-stream"
	}

	if rw, ok := out.(http.ResponseWriter); ok {
		rw.Header().Set("Content-Type", ctype)
		if resp.ContentLength > 0 {
			rw.Header().Set("Content-Length", strconv.FormatInt(resp.ContentLength, 10))
		}
	}

	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", errors.NewUpstream(err.Error())
	}
	return ctype, nil
}

// StreamDownload is the single-step file endpoint: give it an artwork URL (or a
// raw i.pximg.net URL) and it streams the media bytes straight to the client —
// no /download-then-/image round trip. Returns the content-type for the handler.
func StreamDownload(ctx context.Context, targetURL string, out io.Writer) (string, error) {
	// Already a CDN URL? Stream it directly.
	if strings.Contains(targetURL, "i.pximg.net") {
		return StreamImage(ctx, targetURL, out)
	}

	m := reArtworkID.FindStringSubmatch(targetURL)
	if m == nil {
		return "", errors.NewInvalidURL("not a valid Pixiv artwork URL (expected /artworks/{id})")
	}

	md, err := FetchData(ctx, targetURL)
	if err != nil {
		return "", err
	}
	if len(md.Items) == 0 {
		return "", errors.NewUpstream(fmt.Sprintf("no media found for artwork %s", m[1]))
	}
	// First page of the artwork. Ugoira (single video item) streams the zip.
	return StreamImage(ctx, md.Items[0].URL, out)
}
