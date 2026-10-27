package pixiv

import (
	"bufio"
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func loadEnvCookies() {
	f, err := os.Open("../../../.env")
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "PIXIV_COOKIES=") {
			val := strings.TrimPrefix(line, "PIXIV_COOKIES=")
			val = strings.Trim(val, `"'`)
			SetCookies(val)
			break
		}
	}
}

func TestLivePixivWithCookies(t *testing.T) {
	if os.Getenv("LIVE_TEST") == "" {
		t.Skip("skipping live test; set LIVE_TEST=1 to run")
	}

	loadEnvCookies()
	cookies := GetCookies()
	if cookies == "" {
		t.Fatal("PIXIV_COOKIES is empty in .env")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ranks, err := FetchRankings(ctx, "daily", 1)
	if err != nil {
		t.Fatalf("FetchRankings failed: %v", err)
	}
	if len(ranks) == 0 {
		t.Fatal("expected at least 1 ranking item")
	}
	t.Logf("Rankings fetched: %d items. Top #1: %q by %s (PageCount=%d, PageURL=%s)",
		len(ranks), ranks[0].Title, ranks[0].UserName, ranks[0].PageCount, ranks[0].PageURL)

	search, err := SearchArtworks(ctx, "cat", 1, "safe")
	if err != nil {
		t.Fatalf("SearchArtworks failed: %v", err)
	}
	if len(search) == 0 {
		t.Fatal("expected at least 1 search item")
	}
	t.Logf("Search artworks fetched: %d items. First: %q", len(search), search[0].Title)

	// Test FetchData on live artwork
	data, err := FetchData(ctx, "https://www.pixiv.net/artworks/151276944")
	if err != nil {
		t.Fatalf("FetchData failed: %v", err)
	}
	t.Logf("FetchData artwork success: %q by %s, Items: %d", data.Title, data.Owner, len(data.Items))
}
