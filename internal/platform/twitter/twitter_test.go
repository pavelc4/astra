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
