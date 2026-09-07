package facebook

import (
	"strings"
	"testing"
)

// Synthetic bbox mirroring the shape Facebook server-renders for a mixed
// StoryAttachmentAlbumStyleRenderer: count=2, node0 a Video (progressive SD+HD
// + poster), node1 a Photo. A neighbouring album below must NOT be picked up.
// (Real payloads escape slashes as \/; the regex treats the backslash as
// optional, so plain URLs exercise the same path and keep this literal legible.)
const albumMixed = `...garbage...` +
	`"all_subattachments":{"count":2,"nodes":[` +
	`{"deduplication_key":"aaa","media":{"__typename":"Video","is_playable":true,` +
	`"image":{"uri":"https://scontent.test/t15/111111_1_1_n.jpg?ctp=s590x590"},` +
	`"viewer_image":{"uri":"https://scontent.test/t15/111111_1_1_n.jpg?ctp=s478"},` +
	`"video_grid_renderer":{"video":{"videoDeliveryResponseResult":{"progressive_urls":[` +
	`{"progressive_url":"https://scontent.test/vid_sd.mp4","quality":"SD"},` +
	`{"progressive_url":"https://scontent.test/vid_hd.mp4","quality":"HD"}]}}}}},` +
	`{"deduplication_key":"bbb","media":{"__typename":"Photo","is_playable":false,` +
	`"image":{"uri":"https://scontent.test/t39/333333_3_3_n.jpg?ctp=s590x590"},` +
	`"viewer_image":{"height":713,"width":984,"uri":"https://scontent.test/t39/333333_3_3_n.jpg?ctp=s984"}}}` +
	`]}` +
	// neighbouring album further down the page — must NOT be picked up
	`"all_subattachments":{"count":9,"nodes":[{"deduplication_key":"zzz","media":{"__typename":"Photo",` +
	`"viewer_image":{"height":1,"width":1,"uri":"https://scontent.test/t39/999999_9_9_n.jpg?ctp=s1"}}}]}`

func TestExtractAlbumMedia_Mixed(t *testing.T) {
	photos, videos := extractAlbumMedia(albumMixed)

	if len(videos) != 1 {
		t.Fatalf("got %d videos, want 1", len(videos))
	}
	if videos[0].URL != "https://scontent.test/vid_hd.mp4" {
		t.Errorf("video URL = %q, want HD progressive", videos[0].URL)
	}
	if videos[0].Quality != "hd" {
		t.Errorf("video Quality = %q, want hd", videos[0].Quality)
	}
	if videos[0].Thumbnail == nil || !strings.Contains(*videos[0].Thumbnail, "111111") {
		t.Errorf("video poster not set from node image: %v", videos[0].Thumbnail)
	}

	if len(photos) != 1 {
		t.Fatalf("got %d photos, want 1", len(photos))
	}
	if !strings.Contains(photos[0].URL, "333333") {
		t.Errorf("photo URL = %q, want the Photo node", photos[0].URL)
	}
	if photos[0].Quality != "984x713" {
		t.Errorf("photo Quality = %q, want 984x713", photos[0].Quality)
	}
	// The video's poster and the neighbour's photo must never leak into photos.
	for i, p := range photos {
		if strings.Contains(p.URL, "111111") || strings.Contains(p.URL, "vid_") {
			t.Errorf("photo %d is a video poster/MP4: %q", i, p.URL)
		}
		if strings.Contains(p.URL, "999999") {
			t.Errorf("photo %d bled into neighbouring album: %q", i, p.URL)
		}
	}
}

// Pure-photo album: three Photo nodes, count caps and dedupes by fbcdn basename.
const albumPhotosOnly = `...garbage...` +
	`"all_subattachments":{"count":3,"nodes":[` +
	`{"deduplication_key":"1","media":{"__typename":"Photo","image":{"uri":"https://scontent.test/t39/111111_1_1_n.jpg?ctp=s590"},"viewer_image":{"height":1065,"width":799,"uri":"https://scontent.test/t39/111111_1_1_n.jpg?ctp=s799"}}},` +
	`{"deduplication_key":"2","media":{"__typename":"Photo","image":{"uri":"https://scontent.test/t39/222222_2_2_n.jpg?ctp=s590"},"viewer_image":{"height":1066,"width":800,"uri":"https://scontent.test/t39/222222_2_2_n.jpg?ctp=s800"}}},` +
	`{"deduplication_key":"3","media":{"__typename":"Photo","image":{"uri":"https://scontent.test/t39/333333_3_3_n.jpg?ctp=s590"},"viewer_image":{"height":1706,"width":1280,"uri":"https://scontent.test/t39/333333_3_3_n.jpg?ctp=s1280"}}}` +
	`]}`

func TestExtractAlbumMedia_PhotosOnly(t *testing.T) {
	photos, videos := extractAlbumMedia(albumPhotosOnly)
	if len(videos) != 0 {
		t.Fatalf("got %d videos, want 0", len(videos))
	}
	if len(photos) != 3 {
		t.Fatalf("got %d photos, want 3", len(photos))
	}
	// viewer_image (full-res) preferred: quality = WxH, first is 799x1065.
	if photos[0].Quality != "799x1065" {
		t.Errorf("photos[0].Quality = %q, want 799x1065", photos[0].Quality)
	}
	seen := map[string]bool{}
	for i, p := range photos {
		if !strings.HasPrefix(p.URL, "https://scontent.test") || strings.Contains(p.URL, `\/`) {
			t.Errorf("photo %d URL not cleaned: %q", i, p.URL)
		}
		base := photoBaseName(p.URL)
		if base == "" || seen[base] {
			t.Errorf("photo %d bad/dup basename %q", i, base)
		}
		seen[base] = true
	}
}

func TestExtractAlbumMedia_NoAlbum(t *testing.T) {
	if p, v := extractAlbumMedia(`<html>no bbox here</html>`); p != nil || v != nil {
		t.Errorf("want nil,nil for non-album page, got photos=%v videos=%v", p, v)
	}
}

func TestExtractSinglePhoto(t *testing.T) {
	singlePhotoHTML := `"StoryAttachmentPhotoStyleRenderer","viewer_image":{"height":2048,"width":1483},"photo_image":{"uri":"https:\/\/scontent.test\/single.jpg?x=1"}`
	item := extractSinglePhoto(singlePhotoHTML)
	if item == nil {
		t.Fatal("expected single photo item, got nil")
	}
	if item.URL != "https://scontent.test/single.jpg?x=1" {
		t.Errorf("got URL %q, want https://scontent.test/single.jpg?x=1", item.URL)
	}
	if item.Quality != "1483x2048" {
		t.Errorf("got Quality %q, want 1483x2048", item.Quality)
	}
}
