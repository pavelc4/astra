package terabox_test

import (
	"testing"

	"github.com/pavelc4/astra/internal/platform/terabox"
)

func TestTeraBoxURLRegex(t *testing.T) {
	tests := []struct {
		name  string
		url   string
		match bool
	}{
		{
			name:  "standard terabox link",
			url:   "https://terabox.com/s/1abcdef",
			match: true,
		},
		{
			name:  "1024tera link",
			url:   "https://1024tera.com/s/1abcdef",
			match: true,
		},
		{
			name:  "non terabox link",
			url:   "https://google.com",
			match: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := terabox.URLRegex.MatchString(tt.url)
			if got != tt.match {
				t.Errorf("URLRegex.MatchString(%q) = %v, want %v", tt.url, got, tt.match)
			}
		})
	}
}

func TestNonceRegex(t *testing.T) {
	sampleHTML := `<script id="jquery-core-js-extra">var wp = {"nonce":"abc123xyz"};</script>`
	m := terabox.NonceRegex.FindStringSubmatch(sampleHTML)
	if len(m) < 2 || m[1] != "abc123xyz" {
		t.Errorf("expected nonce abc123xyz, got %v", m)
	}
}
