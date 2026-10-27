package pixiv

import (
	"context"
	"sync"
	"testing"
)

func TestReArtworkID(t *testing.T) {
	tests := []struct {
		url      string
		expected string
		match    bool
	}{
		{"https://www.pixiv.net/artworks/12345678", "12345678", true},
		{"https://www.pixiv.net/en/artworks/87654321/", "87654321", true},
		{"https://pixiv.net/artworks/999", "999", true},
		{"https://www.pixiv.net/users/123456", "", false},
		{"invalid-url", "", false},
	}

	for _, tt := range tests {
		m := reArtworkID.FindStringSubmatch(tt.url)
		if tt.match {
			if m == nil || m[1] != tt.expected {
				t.Errorf("reArtworkID(%q) = %v, expected %s", tt.url, m, tt.expected)
			}
		} else {
			if m != nil {
				t.Errorf("reArtworkID(%q) matched unexpectedly: %v", tt.url, m)
			}
		}
	}
}

func TestReUserID(t *testing.T) {
	tests := []struct {
		url      string
		expected string
		match    bool
	}{
		{"https://www.pixiv.net/users/12345678", "12345678", true},
		{"https://www.pixiv.net/en/users/87654321/", "87654321", true},
		{"https://www.pixiv.net/artworks/123456", "", false},
	}

	for _, tt := range tests {
		m := reUserID.FindStringSubmatch(tt.url)
		if tt.match {
			if m == nil || m[1] != tt.expected {
				t.Errorf("reUserID(%q) = %v, expected %s", tt.url, m, tt.expected)
			}
		} else {
			if m != nil {
				t.Errorf("reUserID(%q) matched unexpectedly: %v", tt.url, m)
			}
		}
	}
}

func TestPageURL(t *testing.T) {
	original := "https://i.pximg.net/img-original/img/2023/01/01/00/00/00/12345678_p0.jpg"

	if got := pageURL(0, original); got != original {
		t.Errorf("pageURL(0) = %s, expected %s", got, original)
	}

	expectedP1 := "https://i.pximg.net/img-original/img/2023/01/01/00/00/00/12345678_p1.jpg"
	if got := pageURL(1, original); got != expectedP1 {
		t.Errorf("pageURL(1) = %s, expected %s", got, expectedP1)
	}

	expectedP2 := "https://i.pximg.net/img-original/img/2023/01/01/00/00/00/12345678_p2.jpg"
	if got := pageURL(2, original); got != expectedP2 {
		t.Errorf("pageURL(2) = %s, expected %s", got, expectedP2)
	}
}

func TestCookiesGetterSetter(t *testing.T) {
	const testCookie = "PHPSESSID=test_session_1234"
	SetCookies(testCookie)
	if got := GetCookies(); got != testCookie {
		t.Errorf("GetCookies() = %s, expected %s", got, testCookie)
	}

	// Test concurrent access
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = GetCookies()
		}()
		go func() {
			defer wg.Done()
			SetCookies(testCookie)
		}()
	}
	wg.Wait()
}

func TestFetchDataInvalidURL(t *testing.T) {
	ctx := context.Background()
	_, err := FetchData(ctx, "https://example.com/not-pixiv")
	if err == nil {
		t.Fatal("expected error for invalid pixiv url, got nil")
	}
}
