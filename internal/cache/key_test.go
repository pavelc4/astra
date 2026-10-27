package cache_test

import (
	"testing"

	"github.com/pavelc4/astra/internal/cache"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty url",
			input:    "   ",
			expected: "",
		},
		{
			name:     "instagram with tracking and trailing slash",
			input:    "http://www.instagram.com/reel/C7xyz/?igsh=abc1234&utm_source=ig_web_copy_link#fragment",
			expected: "https://instagram.com/reel/C7xyz",
		},
		{
			name:     "mobile facebook with tracking",
			input:    "http://m.facebook.com/watch/?v=123456&mibextid=123",
			expected: "https://facebook.com/watch?v=123456",
		},
		{
			name:     "x.com domain aliased to twitter.com with s parameter",
			input:    "https://x.com/user/status/987654321?s=20&t=abcdef",
			expected: "https://twitter.com/user/status/987654321",
		},
		{
			name:     "tiktok with is_from_webapp and sender_device",
			input:    "https://www.tiktok.com/@creator/video/7123456789012345678?is_from_webapp=1&sender_device=pc",
			expected: "https://tiktok.com/@creator/video/7123456789012345678",
		},
		{
			name:     "spotify with si track tracking",
			input:    "https://open.spotify.com/track/4cOdK2wGLETKBW3PvgPWqT?si=0123456789abcdef",
			expected: "https://open.spotify.com/track/4cOdK2wGLETKBW3PvgPWqT",
		},
		{
			name:     "root url preserved without breaking",
			input:    "https://instagram.com/",
			expected: "https://instagram.com/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cache.NormalizeURL(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizeURL(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestHashKey(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		url       string
	}{
		{
			name:      "instagram download",
			operation: "instagram_download",
			url:       "https://instagram.com/p/123",
		},
		{
			name:      "tiktok download",
			operation: "tiktok_download",
			url:       "https://tiktok.com/@user/video/456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h1 := cache.HashKey(tt.operation, tt.url)
			h2 := cache.HashKey(tt.operation, tt.url)

			if len(h1) != 64 {
				t.Fatalf("expected 64-char sha256 hex, got len=%d (%s)", len(h1), h1)
			}
			if h1 != h2 {
				t.Errorf("hash must be deterministic: %q != %q", h1, h2)
			}

			// Different operation produces different hash
			diffOp := cache.HashKey(tt.operation+"_diff", tt.url)
			if diffOp == h1 {
				t.Errorf("different operation should produce different hash")
			}
		})
	}
}

func BenchmarkNormalizeURL(b *testing.B) {
	raw := "http://www.instagram.com/reel/C7xyz/?igsh=abc1234&utm_source=ig_web_copy_link#fragment"
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = cache.NormalizeURL(raw)
	}
}

func BenchmarkHashKey(b *testing.B) {
	u := "https://instagram.com/reel/C7xyz"
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = cache.HashKey("instagram_download", u)
	}
}
