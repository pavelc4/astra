package twitter_test

import (
	"testing"

	"github.com/pavelc4/astra/internal/platform/twitter"
)

func TestTwitterURLRegex(t *testing.T) {
	tests := []struct {
		name  string
		url   string
		match bool
	}{
		{
			name:  "standard twitter URL",
			url:   "https://twitter.com/user/status/1234567890",
			match: true,
		},
		{
			name:  "x.com URL",
			url:   "https://x.com/user/status/1234567890",
			match: true,
		},
		{
			name:  "non twitter URL",
			url:   "https://instagram.com/reel/123",
			match: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := twitter.URLRegex.MatchString(tt.url)
			if got != tt.match {
				t.Errorf("URLRegex.MatchString(%q) = %v, want %v", tt.url, got, tt.match)
			}
		})
	}
}

func TestQualityRegex(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Download MP4 (720p)", "720p"},
		{"Download MP4 (1080p)", "1080p"},
		{"No quality here", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			m := twitter.QualityRegex.FindStringSubmatch(tt.input)
			got := ""
			if len(m) > 1 {
				got = m[1]
			}
			if got != tt.want {
				t.Errorf("QualityRegex.Find(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseResult_OmitThumbnailWhenVideo(t *testing.T) {
	htmlData := `
	<div class="tw-video">
		<input id="TwitterId" value="123456" />
		<div class="tw-middle">
			<h3>Sample Twitter Video</h3>
			<p>0:45</p>
		</div>
		<div class="thumbnail">
			<img src="https://pbs.twimg.com/amplify_video_thumb/123/img/thumb.jpg" />
		</div>
		<a class="tw-button-dl" href="https://dl.snapcdn.app/get?token=thumb">Render / Download Photo</a>
		<a class="tw-button-dl" href="https://dl.snapcdn.app/get?token=v1080">Download MP4 (1080p)</a>
		<a class="tw-button-dl" href="https://dl.snapcdn.app/get?token=v720">Download MP4 (720p)</a>
	</div>
	`

	res := twitter.ParseResult([]byte(htmlData))
	if res.Type != "video" {
		t.Fatalf("expected res.Type to be 'video', got %q", res.Type)
	}

	if res.Thumbnail == nil || *res.Thumbnail != "https://pbs.twimg.com/amplify_video_thumb/123/img/thumb.jpg" {
		t.Errorf("expected thumbnail to be preserved, got %v", res.Thumbnail)
	}

	for _, d := range res.Downloads {
		if d.Type == "image" {
			t.Errorf("expected no image download item when video is present, found %+v", d)
		}
	}

	if len(res.Downloads) != 1 {
		t.Fatalf("expected 1 best video download, got %d", len(res.Downloads))
	}

	if res.Downloads[0].Quality != "1080p" {
		t.Errorf("expected best quality 1080p, got %s", res.Downloads[0].Quality)
	}
}

func TestParseResult_MultiVideo(t *testing.T) {
	htmlData := `
	<div class="tw-video">
		<div class="thumbnail"><img src="https://pbs.twimg.com/amplify_video_thumb/1/img/t1.jpg" /></div>
		<div class="tw-right"><div class="dl-action">
			<p><a class="tw-button-dl" href="https://dl.snapcdn.app/get?token=v1_1080">Download MP4 (1080p)</a></p>
			<p><a class="tw-button-dl" href="https://dl.snapcdn.app/get?token=v1_720">Download MP4 (720p)</a></p>
			<p><a class="tw-button-dl" href="https://dl.snapcdn.app/get?token=t1">Download Photo</a></p>
		</div></div>
	</div>
	<div class="tw-video">
		<div class="thumbnail"><img src="https://pbs.twimg.com/amplify_video_thumb/2/img/t2.jpg" /></div>
		<div class="tw-right"><div class="dl-action">
			<p><a class="tw-button-dl" href="https://dl.snapcdn.app/get?token=v2_720">Download MP4 (720p)</a></p>
			<p><a class="tw-button-dl" href="https://dl.snapcdn.app/get?token=v2_480">Download MP4 (480p)</a></p>
			<p><a class="tw-button-dl" href="https://dl.snapcdn.app/get?token=t2">Download Photo</a></p>
		</div></div>
	</div>
	<div class="tw-video">
		<div class="thumbnail"><img src="https://pbs.twimg.com/amplify_video_thumb/3/img/t3.jpg" /></div>
		<div class="tw-right"><div class="dl-action">
			<p><a class="tw-button-dl" href="https://dl.snapcdn.app/get?token=v3_1080">Download MP4 (1080p)</a></p>
			<p><a class="tw-button-dl" href="https://dl.snapcdn.app/get?token=t3">Download Photo</a></p>
		</div></div>
	</div>
	<input type="hidden" id="TwitterId" value="2115271858660765981" />
	`

	res := twitter.ParseResult([]byte(htmlData))
	if res.Type != "video" {
		t.Fatalf("expected res.Type to be 'video', got %q", res.Type)
	}

	if len(res.Downloads) != 3 {
		t.Fatalf("expected 3 video downloads, got %d", len(res.Downloads))
	}

	for i, d := range res.Downloads {
		if d.Type != "video" {
			t.Errorf("item %d: expected video type, got %s", i, d.Type)
		}
	}

	if res.Downloads[0].Quality != "1080p" || res.Downloads[1].Quality != "720p" || res.Downloads[2].Quality != "1080p" {
		t.Errorf("unexpected qualities: %v, %v, %v", res.Downloads[0].Quality, res.Downloads[1].Quality, res.Downloads[2].Quality)
	}
}

func TestParseResult_MixedMedia(t *testing.T) {
	htmlData := `
	<div class="tw-video">
		<div class="thumbnail">
			<div class="image-tw open-popup photo-tw">
				<img src="https://pbs.twimg.com/media/HVmX1qRaUAAzer7.jpg">
			</div>
		</div>
		<div class="tw-right"><div class="dl-action">
			<p><a class="tw-button-dl" href="https://dl.snapcdn.app/get?token=photo1">Download Photo</a></p>
		</div></div>
		<div class="tw-middle"><h3>Mixed post</h3></div>
	</div>
	<div class="tw-video">
		<div class="thumbnail">
			<img src="https://pbs.twimg.com/media/HVmX1qRaUAAzer7.jpg">
		</div>
		<div class="tw-right"><div class="dl-action">
			<p><a class="tw-button-dl" href="https://dl.snapcdn.app/get?token=v720">Download MP4 (720p)</a></p>
			<p><a class="tw-button-dl" href="https://dl.snapcdn.app/get?token=v360">Download MP4 (360p)</a></p>
			<p><a class="tw-button-dl" href="https://dl.snapcdn.app/get?token=thumb">Download Photo</a></p>
		</div></div>
		<div class="tw-middle"><p>0:42</p></div>
	</div>
	<input type="hidden" id="TwitterId" value="2114891265670365220" />
	`

	res := twitter.ParseResult([]byte(htmlData))
	if res.Type != "multiple" {
		t.Fatalf("expected res.Type to be 'multiple', got %q", res.Type)
	}

	if len(res.Downloads) != 2 {
		t.Fatalf("expected 2 downloads (1 photo + 1 video), got %d", len(res.Downloads))
	}

	if res.Downloads[0].Type != "image" {
		t.Errorf("expected download 0 to be image, got %s", res.Downloads[0].Type)
	}
	if res.Downloads[1].Type != "video" {
		t.Errorf("expected download 1 to be video, got %s", res.Downloads[1].Type)
	}
	if res.Downloads[1].Quality != "720p" {
		t.Errorf("expected video quality 720p, got %s", res.Downloads[1].Quality)
	}
}

func TestParseResult_PreservePhotosWhenNoVideo(t *testing.T) {
	htmlData := `
	<div class="tw-video">
		<input id="TwitterId" value="789012" />
		<div class="tw-middle">
			<h3>Sample Twitter Photo</h3>
		</div>
		<a class="tw-button-dl" href="https://dl.snapcdn.app/get?token=photo1">Download Photo</a>
	</div>
	`

	res := twitter.ParseResult([]byte(htmlData))
	if res.Type != "photo" {
		t.Fatalf("expected res.Type to be 'photo', got %q", res.Type)
	}

	if len(res.Downloads) != 1 {
		t.Fatalf("expected 1 photo download, got %d", len(res.Downloads))
	}

	if res.Downloads[0].Type != "image" {
		t.Errorf("expected download type to be 'image', got %q", res.Downloads[0].Type)
	}
}
