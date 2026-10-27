package spotify_test

import (
	"testing"

	"github.com/pavelc4/astra/internal/platform/spotify"
)

func TestSpotifyURLRegex(t *testing.T) {
	tests := []struct {
		name  string
		url   string
		match bool
	}{
		{
			name:  "standard spotify track URL",
			url:   "https://open.spotify.com/track/4cOdK2wGLETKBW3PvgPWqT",
			match: true,
		},
		{
			name:  "spotify playlist URL",
			url:   "https://open.spotify.com/playlist/37i9dQZF1DXcBWIGoYBM5M",
			match: true,
		},
		{
			name:  "non spotify URL",
			url:   "https://soundcloud.com/artist/track",
			match: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := spotify.URLRegex.MatchString(tt.url)
			if got != tt.match {
				t.Errorf("URLRegex.MatchString(%q) = %v, want %v", tt.url, got, tt.match)
			}
		})
	}
}

func TestConstants(t *testing.T) {
	if spotify.PlatformName != "spotify" {
		t.Errorf("expected platform name spotify, got %s", spotify.PlatformName)
	}
}
