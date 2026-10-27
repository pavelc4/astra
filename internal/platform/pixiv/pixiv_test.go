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

func TestFetchUgoiraDirectInvalidURL(t *testing.T) {
	ctx := context.Background()
	_, err := FetchUgoiraDirect(ctx, "https://example.com/not-pixiv")
	if err == nil {
		t.Fatal("expected error for invalid ugoira artwork url, got nil")
	}
}

func TestFetchUserAllWorksInvalidURL(t *testing.T) {
	ctx := context.Background()
	_, err := FetchUserAllWorks(ctx, "https://example.com/not-user")
	if err == nil {
		t.Fatal("expected error for invalid user url, got nil")
	}
}

func TestValidRankingModes(t *testing.T) {
	expectedModes := []string{
		"daily", "weekly", "monthly", "rookie", "original",
		"male", "female", "daily_r18", "weekly_r18", "r18g",
	}

	for _, m := range expectedModes {
		if !validRankingModes[m] {
			t.Errorf("expected ranking mode %q to be valid", m)
		}
	}
}

func TestReNovelID(t *testing.T) {
	tests := []struct {
		url      string
		expected string
		match    bool
	}{
		{"https://www.pixiv.net/novel/show.php?id=12345678", "12345678", true},
		{"https://www.pixiv.net/en/novel/show.php?id=87654321", "87654321", true},
		{"https://www.pixiv.net/novels/11223344", "11223344", true},
		{"https://www.pixiv.net/novel/556677", "556677", true},
		{"https://pixiv.net/artworks/999", "999", true},
		{"https://www.pixiv.net/users/123456", "", false},
		{"invalid-novel-url", "", false},
	}

	for _, tt := range tests {
		m := reNovelID.FindStringSubmatch(tt.url)
		if tt.match {
			if m == nil || m[1] != tt.expected {
				t.Errorf("reNovelID(%q) = %v, expected %s", tt.url, m, tt.expected)
			}
		} else {
			if m != nil {
				t.Errorf("reNovelID(%q) matched unexpectedly: %v", tt.url, m)
			}
		}
	}
}

func TestFetchNovelInvalidURL(t *testing.T) {
	ctx := context.Background()
	_, err := FetchNovel(ctx, "https://example.com/not-novel")
	if err == nil {
		t.Fatal("expected error for invalid pixiv novel url, got nil")
	}
}

func TestSearchArtworksValidation(t *testing.T) {
	ctx := context.Background()
	_, err := SearchArtworks(ctx, "", 1, "all")
	if err == nil {
		t.Fatal("expected error for empty search keyword, got nil")
	}
}

func TestFetchUserDetailInvalidURL(t *testing.T) {
	ctx := context.Background()
	_, err := FetchUserDetail(ctx, "https://example.com/not-user")
	if err == nil {
		t.Fatal("expected error for invalid pixiv user url, got nil")
	}
}
