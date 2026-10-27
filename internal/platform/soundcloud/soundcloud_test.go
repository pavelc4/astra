package soundcloud_test

import (
	"testing"

	"github.com/pavelc4/astra/internal/platform/soundcloud"
)

func TestSoundCloudURLRegex(t *testing.T) {
	tests := []struct {
		name  string
		url   string
		match bool
	}{
		{
			name:  "standard soundcloud URL",
			url:   "https://soundcloud.com/artist/cool-track",
			match: true,
		},
		{
			name:  "short on.soundcloud.com URL",
			url:   "https://on.soundcloud.com/abcdef",
			match: true,
		},
		{
			name:  "non soundcloud URL",
			url:   "https://open.spotify.com/track/123",
			match: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := soundcloud.URLRegex.MatchString(tt.url)
			if got != tt.match {
				t.Errorf("URLRegex.MatchString(%q) = %v, want %v", tt.url, got, tt.match)
			}
		})
	}
}

func TestConstants(t *testing.T) {
	if soundcloud.PlatformName != "soundcloud" {
		t.Errorf("expected platform name soundcloud, got %s", soundcloud.PlatformName)
	}
}
