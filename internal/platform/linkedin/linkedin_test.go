package linkedin_test

import (
	"testing"

	"github.com/pavelc4/astra/internal/platform/linkedin"
)

func TestLinkedInURLRegex(t *testing.T) {
	tests := []struct {
		name  string
		url   string
		match bool
	}{
		{
			name:  "standard linkedin post URL",
			url:   "https://www.linkedin.com/posts/username_activity-1234567890-abcd",
			match: true,
		},
		{
			name:  "non linkedin URL",
			url:   "https://twitter.com/user/status/123",
			match: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := linkedin.URLRegex.MatchString(tt.url)
			if got != tt.match {
				t.Errorf("URLRegex.MatchString(%q) = %v, want %v", tt.url, got, tt.match)
			}
		})
	}
}
