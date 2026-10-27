package capcut_test

import (
	"testing"

	"github.com/pavelc4/astra/internal/platform/capcut"
)

func TestCapCutURLRegex(t *testing.T) {
	tests := []struct {
		name  string
		url   string
		match bool
	}{
		{
			name:  "standard capcut link",
			url:   "https://www.capcut.com/template-detail/12345",
			match: true,
		},
		{
			name:  "non capcut link",
			url:   "https://youtube.com/watch?v=123",
			match: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := capcut.URLRegex.MatchString(tt.url)
			if got != tt.match {
				t.Errorf("URLRegex.MatchString(%q) = %v, want %v", tt.url, got, tt.match)
			}
		})
	}
}
