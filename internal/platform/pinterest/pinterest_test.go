package pinterest_test

import (
	"testing"

	"github.com/pavelc4/astra/internal/platform/pinterest"
)

func TestPinterestURLRegex(t *testing.T) {
	tests := []struct {
		name  string
		url   string
		match bool
	}{
		{
			name:  "standard pinterest pin URL",
			url:   "https://www.pinterest.com/pin/123456789/",
			match: true,
		},
		{
			name:  "pin.it shortlink",
			url:   "https://pin.it/7abcdef",
			match: true,
		},
		{
			name:  "non pinterest URL",
			url:   "https://instagram.com/p/123",
			match: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pinterest.URLRegex.MatchString(tt.url)
			if got != tt.match {
				t.Errorf("URLRegex.MatchString(%q) = %v, want %v", tt.url, got, tt.match)
			}
		})
	}
}

func TestPinIDRegex(t *testing.T) {
	url := "https://www.pinterest.com/pin/9876543210/"
	m := pinterest.PinIDRegex.FindStringSubmatch(url)
	if len(m) < 2 || m[1] != "9876543210" {
		t.Errorf("expected pin ID 9876543210, got %v", m)
	}
}
