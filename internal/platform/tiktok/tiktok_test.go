package tiktok_test

import (
	"testing"

	"github.com/pavelc4/astra/internal/platform/tiktok"
)

func TestTikTokURLRegex(t *testing.T) {
	tests := []struct {
		name  string
		url   string
		match bool
	}{
		{
			name:  "standard desktop web URL",
			url:   "https://www.tiktok.com/@creator/video/7123456789012345678",
			match: true,
		},
		{
			name:  "vt shortlink",
			url:   "https://vt.tiktok.com/ZS8XXXXXX/",
			match: true,
		},
		{
			name:  "vm shortlink",
			url:   "https://vm.tiktok.com/ZM8XXXXXX/",
			match: true,
		},
		{
			name:  "non tiktok url",
			url:   "https://instagram.com/reel/123",
			match: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tiktok.URLRegex.MatchString(tt.url)
			if got != tt.match {
				t.Errorf("URLRegex.MatchString(%q) = %v, want %v", tt.url, got, tt.match)
			}
		})
	}
}

func TestConstants(t *testing.T) {
	if tiktok.PlatformName != "tiktok" {
		t.Errorf("expected PlatformName tiktok, got %s", tiktok.PlatformName)
	}
	if tiktok.BaseAPIURL == "" {
		t.Errorf("expected non-empty BaseAPIURL")
	}
}
