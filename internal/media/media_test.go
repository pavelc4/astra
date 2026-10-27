package media

import (
	"encoding/json"
	"testing"
)

// oldResult is the exact struct the Downloads-family platforms used before the
// refactor (reddit/spotify/pinterest/terabox/soundcloud). The test proves
// media.Downloads marshals byte-for-byte identically, so the HTTP API response
// is unchanged.
type oldResult struct {
	Platform  string `json:"platform"`
	Title     string `json:"title"`
	Thumbnail string `json:"thumbnail,omitempty"`
	Downloads []Item `json:"downloads"`
}

func TestDownloadsByteEqual(t *testing.T) {
	items := []Item{
		{Label: "Video 720p", URL: "https://x/v.mp4", Type: Video, Quality: "720p"},
		{Label: "Image original", URL: "https://x/i.jpg", Type: Image, Quality: "original"},
	}

	cases := []struct {
		name             string
		platform, title  string
		thumb            string
		items            []Item
	}{
		{"full", "reddit", "A post", "https://x/thumb.jpg", items},
		{"no-thumbnail", "spotify", "Track", "", items[:1]},
		{"nil-items", "terabox", "Folder", "", nil},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			old, err := json.Marshal(oldResult{c.platform, c.title, c.thumb, c.items})
			if err != nil {
				t.Fatal(err)
			}
			got, err := json.Marshal(Downloads(c.platform, c.title, c.thumb, c.items))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(old) {
				t.Errorf("byte mismatch:\n old: %s\n new: %s", old, got)
			}
		})
	}
}


// old meta-family structs (Facebook/Instagram) — proves media.Meta reproduces
// both envelopes byte-for-byte from one shape.
type oldMediaItem struct {
	Quality   string  `json:"quality"`
	Thumbnail *string `json:"thumbnail,omitempty"`
	URL       string  `json:"url"`
}
type oldFacebookResult struct {
	Platform string         `json:"platform"`
	Caption  string         `json:"caption,omitempty"`
	Duration string         `json:"duration,omitempty"`
	Audio    string         `json:"audio,omitempty"`
	Photos   []oldMediaItem `json:"photos,omitempty"`
	Videos   []oldMediaItem `json:"videos,omitempty"`
	Raw      []oldMediaItem `json:"raw,omitempty"`
}
type oldInstagramResult struct {
	Platform string         `json:"platform"`
	Caption  string         `json:"caption,omitempty"`
	Owner    string         `json:"owner,omitempty"`
	Audio    string         `json:"audio,omitempty"`
	Photos   []oldMediaItem `json:"photos,omitempty"`
	Videos   []oldMediaItem `json:"videos,omitempty"`
	Raw      []oldMediaItem `json:"raw,omitempty"`
}

func TestMetaByteEqual_Facebook(t *testing.T) {
	thumb := "https://x/thumb.jpg"
	items := []Item{
		{URL: "https://x/v1.mp4", Type: Video, Quality: "hd", Thumbnail: thumb},
		{URL: "https://x/p1.jpg", Type: Image, Quality: "799x1065"},
		{URL: "https://x/p2.jpg", Type: Image, Quality: "800x1066"},
	}
	got, _ := json.Marshal(Meta("facebook", "cap", items, WithDuration("20")))

	old, _ := json.Marshal(oldFacebookResult{
		Platform: "facebook", Caption: "cap", Duration: "20",
		Videos: []oldMediaItem{{Quality: "hd", Thumbnail: &thumb, URL: "https://x/v1.mp4"}},
		Photos: []oldMediaItem{{Quality: "799x1065", URL: "https://x/p1.jpg"}, {Quality: "800x1066", URL: "https://x/p2.jpg"}},
		Raw: []oldMediaItem{
			{Quality: "hd", Thumbnail: &thumb, URL: "https://x/v1.mp4"},
			{Quality: "799x1065", URL: "https://x/p1.jpg"},
			{Quality: "800x1066", URL: "https://x/p2.jpg"},
		},
	})
	if string(got) != string(old) {
		t.Errorf("facebook mismatch:\n old: %s\n new: %s", old, got)
	}
}

func TestMetaByteEqual_Instagram(t *testing.T) {
	items := []Item{
		{URL: "https://x/v.mp4", Type: Video, Quality: "hd"},
		{URL: "https://x/p.jpg", Type: Image, Quality: ""},
	}
	got, _ := json.Marshal(Meta("instagram", "cap", items, WithOwner("alice"), WithAudio("a.mp3")))

	old, _ := json.Marshal(oldInstagramResult{
		Platform: "instagram", Caption: "cap", Owner: "alice", Audio: "a.mp3",
		Videos: []oldMediaItem{{Quality: "hd", URL: "https://x/v.mp4"}},
		Photos: []oldMediaItem{{Quality: "", URL: "https://x/p.jpg"}},
		Raw: []oldMediaItem{
			{Quality: "hd", URL: "https://x/v.mp4"},
			{Quality: "", URL: "https://x/p.jpg"},
		},
	})
	if string(got) != string(old) {
		t.Errorf("instagram mismatch:\n old: %s\n new: %s", old, got)
	}
}
