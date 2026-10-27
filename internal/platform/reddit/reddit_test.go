package reddit_test

import (
	"testing"

	"github.com/pavelc4/astra/internal/platform/reddit"
)

func TestRedditURLRegex(t *testing.T) {
	tests := []struct {
		name  string
		url   string
		match bool
	}{
		{
			name:  "standard reddit URL",
			url:   "https://www.reddit.com/r/golang/comments/123/cool_post/",
			match: true,
		},
		{
			name:  "v.redd.it direct media",
			url:   "https://v.redd.it/abcdef12345",
			match: true,
		},
		{
			name:  "i.redd.it direct image",
			url:   "https://i.redd.it/abcdef12345.jpg",
			match: true,
		},
		{
			name:  "non reddit URL",
			url:   "https://tiktok.com/@user/123",
			match: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reddit.URLRegex.MatchString(tt.url)
			if got != tt.match {
				t.Errorf("URLRegex.MatchString(%q) = %v, want %v", tt.url, got, tt.match)
			}
		})
	}
}
