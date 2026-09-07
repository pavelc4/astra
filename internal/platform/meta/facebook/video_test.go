package facebook

import (
	"strings"
	"testing"
)

// Synthetic slice mirroring the progressive_urls block Facebook server-renders
// into a watch page (HD + SD), plus a message.text caption. Escaped slashes
// match the real payload.
const watchPage = `<title>Video</title>` +
	`"message":{"text":"my reel caption"}` +
	`"progressive_urls":[` +
	`{"progressive_url":"https:\/\/scontent.test\/sd.mp4?x=1","metadata":{"quality":"SD"}},` +
	`{"progressive_url":"https:\/\/scontent.test\/hd.mp4?x=1","metadata":{"quality":"HD"}}` +
	`]`

func TestExtractProgressiveVideos(t *testing.T) {
	vids := extractProgressiveVideos(watchPage)
	if len(vids) != 2 {
		t.Fatalf("got %d videos, want 2 (HD+SD)", len(vids))
	}
	if vids[0].Quality != "hd" { // HD first
		t.Errorf("vids[0].Quality = %q, want hd", vids[0].Quality)
	}
	if vids[1].Quality != "sd" {
		t.Errorf("vids[1].Quality = %q, want sd", vids[1].Quality)
	}
	for i, v := range vids {
		if !strings.HasPrefix(v.URL, "https://scontent.test") || !strings.Contains(v.URL, ".mp4") || strings.Contains(v.URL, `\/`) {
			t.Errorf("video %d URL not a clean mp4: %q", i, v.URL)
		}
	}
}

func TestExtractProgressiveVideos_None(t *testing.T) {
	if got := extractProgressiveVideos(`<html>no video</html>`); got != nil {
		t.Errorf("want nil, got %v", got)
	}
}

// DASH-only video: no progressive_urls, MP4s only in browser_native_*. This is
// the reel / group-video case that used to fail or fall back to a poster image.
func TestExtractBrowserNativeVideos(t *testing.T) {
	// SD before HD in the source; extractor must still emit HD first.
	page := `"browser_native_sd_url":"https:\/\/video.test\/sd.mp4?x=1"` +
		`,"browser_native_hd_url":"https:\/\/video.test\/hd.mp4?x=1"`
	if got := extractProgressiveVideos(page); got != nil {
		t.Fatalf("progressive should be nil for DASH-only page, got %v", got)
	}
	vids := extractBrowserNativeVideos(page)
	if len(vids) != 2 {
		t.Fatalf("got %d videos, want 2 (hd+sd)", len(vids))
	}
	if vids[0].Quality != "hd" || !strings.Contains(vids[0].URL, "hd.mp4") {
		t.Errorf("vids[0] = %+v, want clean hd mp4 first", vids[0])
	}
	if vids[1].Quality != "sd" || !strings.Contains(vids[1].URL, "sd.mp4") {
		t.Errorf("vids[1] = %+v, want clean sd mp4", vids[1])
	}
	for _, v := range vids {
		if strings.Contains(v.URL, `\/`) {
			t.Errorf("URL not cleaned: %q", v.URL)
		}
	}
	if got := extractBrowserNativeVideos(`<html>no video</html>`); got != nil {
		t.Errorf("want nil for page with no browser_native, got %v", got)
	}
}

func TestIsVideoPage(t *testing.T) {
	if !isVideoPage(`<meta property="og:type" content="video.other" />`) {
		t.Error("video.other should be a video page")
	}
	if isVideoPage(`<meta property="og:type" content="article" />`) {
		t.Error("article should not be a video page")
	}
	if isVideoPage(`<html>no og:type</html>`) {
		t.Error("missing og:type should not be a video page")
	}
	// Facebook group pages render og:type="video.other" for all posts; must not be treated as video pages.
	if isVideoPage(`<meta property="og:type" content="video.other" /><meta name="application-name" content="Grup Facebook" />`) {
		t.Error("group page with video.other should not be a video page")
	}
	if isVideoPage(`<meta property="og:type" content="video.other" /><meta property="og:url" content="https://www.facebook.com/groups/test/posts/123/" />`) {
		t.Error("group url with video.other should not be a video page")
	}
}

func TestExtractProgressiveVideos_SkipAds(t *testing.T) {
	// A page containing an ad video followed by a legitimate progressive video in a separate story.
	adPage := `"StoryAttachmentLinkOpenCallToActionRenderer","link_style":"VIDEO_DR_STYLE"` +
		`,"progressive_urls":[{"progressive_url":"https:\/\/scontent.test\/ad.mp4","metadata":{"quality":"HD"}}]` +
		strings.Repeat(" ", 3000) +
		`"other_story":{"progressive_urls":[{"progressive_url":"https:\/\/scontent.test\/real.mp4","metadata":{"quality":"HD"}}]}`

	vids := extractProgressiveVideos(adPage)
	if len(vids) != 1 {
		t.Fatalf("got %d videos, want 1 (ad skipped)", len(vids))
	}
	if vids[0].URL != "https://scontent.test/real.mp4" {
		t.Errorf("got %q, want https://scontent.test/real.mp4", vids[0].URL)
	}
}

func TestExtractVideoCaption(t *testing.T) {
	// message.text wins over the generic <title>Video</title>.
	if got := extractVideoCaption(watchPage); got != "my reel caption" {
		t.Errorf("caption = %q, want %q", got, "my reel caption")
	}
	// falls back to <title> when no message.text.
	if got := extractVideoCaption(`<title>Fallback</title>`); got != "Fallback" {
		t.Errorf("fallback caption = %q, want Fallback", got)
	}
}

func TestCleanJSURL(t *testing.T) {
	// escaped slashes, & (&) and % (%), and &amp; all normalize.
	got := cleanJSURL(`https:\/\/x\/a.mp4?_nc_cat=1&oh=2%ff&amp;z=3`)
	want := "https://x/a.mp4?_nc_cat=1&oh=2%ff&z=3"
	if got != want {
		t.Errorf("cleanJSURL:\n got %q\n want %q", got, want)
	}
}
