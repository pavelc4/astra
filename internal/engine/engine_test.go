package engine

import (
	"context"
	"testing"

	"github.com/pavelc4/astra/internal/media"
)

func TestDefaultRegistry_Resolve(t *testing.T) {
	reg := DefaultRegistry()

	tests := []struct {
		url          string
		expectedName string
		shouldMatch  bool
	}{
		// TikTok
		{"https://www.tiktok.com/@user/video/123456", "tiktok", true},
		{"https://vt.tiktok.com/ZS2xyz/", "tiktok", true},
		{"https://vm.tiktok.com/ZS2xyz/", "tiktok", true},
		{"https://douyin.com/video/123", "tiktok", true},

		// Twitter / X
		{"https://twitter.com/jack/status/20", "twitter", true},
		{"https://x.com/user/status/123456", "twitter", true},
		{"https://t.co/shortlink", "twitter", true},

		// Instagram
		{"https://www.instagram.com/p/DAxyz123/", "instagram", true},
		{"https://instagr.am/reel/DAxyz123/", "instagram", true},

		// Facebook
		{"https://www.facebook.com/watch/?v=12345", "facebook", true},
		{"https://fb.watch/shortlink/", "facebook", true},
		{"https://fb.com/user/videos/123", "facebook", true},

		// Threads
		{"https://www.threads.net/@user/post/DAxyz", "threads", true},
		{"https://threads.com/@user/post/DAxyz", "threads", true},

		// Reddit
		{"https://www.reddit.com/r/golang/comments/123456/post/", "reddit", true},
		{"https://redd.it/123456", "reddit", true},

		// Pinterest
		{"https://www.pinterest.com/pin/123456789/", "pinterest", true},
		{"https://pin.it/7abcXYZ", "pinterest", true},
		{"https://pinterest.co.uk/pin/123/", "pinterest", true},

		// TeraBox
		{"https://terabox.com/s/1abcdef", "terabox", true},
		{"https://1024tera.com/s/1abcdef", "terabox", true},
		{"https://teraboxapp.com/s/1abcdef", "terabox", true},

		// Spotify
		{"https://open.spotify.com/track/4cOdK2wGLETKBW3PvgPWqT", "spotify", true},
		{"https://spoti.fi/3xyz", "spotify", true},

		// SoundCloud
		{"https://soundcloud.com/artist/track-title", "soundcloud", true},
		{"https://on.soundcloud.com/abc12", "soundcloud", true},

		// CapCut
		{"https://www.capcut.com/template-detail/12345678", "capcut", true},

		// LinkedIn
		{"https://www.linkedin.com/posts/username_activity-123456", "linkedin", true},
		{"https://lnkd.in/gAbCdEf", "linkedin", true},

		// Pixiv
		{"https://www.pixiv.net/artworks/12345678", "pixiv", true},

		// Unsupported
		{"https://example.com/video/123", "", false},
		{"https://youtube.com/watch?v=123", "", false},
		{"not-a-valid-url", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			ext, err := reg.Resolve(tt.url)
			if tt.shouldMatch {
				if err != nil {
					t.Fatalf("expected to resolve %s, got error: %v", tt.url, err)
				}
				if ext.Name() != tt.expectedName {
					t.Errorf("expected extractor %s, got %s", tt.expectedName, ext.Name())
				}
			} else {
				if err == nil {
					t.Fatalf("expected error for %s, but resolved to %s", tt.url, ext.Name())
				}
			}
		})
	}
}

type dummyExtractor struct {
	name string
	host string
}

func (d *dummyExtractor) Name() string { return d.name }
func (d *dummyExtractor) Matches(rawURL string) bool {
	return matchHost(rawURL, func(h string) bool { return h == d.host })
}
func (d *dummyExtractor) Extract(ctx context.Context, rawURL string) (any, error) {
	return media.Downloads(d.name, "Test Title", "https://example.com/thumb.jpg", nil), nil
}

func TestCustomExtractorRegistration(t *testing.T) {
	reg := NewRegistry()
	if len(reg.Extractors()) != 0 {
		t.Fatalf("expected empty registry, got %d extractors", len(reg.Extractors()))
	}

	reg.Register(&dummyExtractor{name: "custom_platform", host: "custom.org"})

	exts := reg.Extractors()
	if len(exts) != 1 {
		t.Fatalf("expected 1 extractor, got %d", len(exts))
	}

	resolved, err := reg.Resolve("https://custom.org/item/1")
	if err != nil {
		t.Fatalf("expected resolve to succeed: %v", err)
	}
	if resolved.Name() != "custom_platform" {
		t.Errorf("expected custom_platform, got %s", resolved.Name())
	}

	rawRes, err := resolved.Extract(context.Background(), "https://custom.org/item/1")
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	res, ok := rawRes.(*media.Media)
	if !ok || res.Platform != "custom_platform" {
		t.Errorf("expected platform custom_platform, got %v", rawRes)
	}
}
