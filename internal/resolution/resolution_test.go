package resolution

import (
	"testing"

	"github.com/pavelc4/astra/internal/media"
)

func TestRank(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"4K", 2160},
		{"2K", 1440},
		{"1080p", 1080},
		{"720p", 720},
		{"480p", 480},
		{"360p", 360},
		{"240p", 240},
		{"hd", 1080},
		{"sd", 480},
		{"original", 9999},
		{"1920x1080", 1080},
		{"720x1280", 720},
		{"720w", 720},
		{"", 0},
		{"unknown", 0},
	}

	for _, tt := range tests {
		got := Rank(tt.input)
		if got != tt.expected {
			t.Errorf("Rank(%q) = %d, expected %d", tt.input, got, tt.expected)
		}
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://v.redd.it/cmaf_1080.mp4", P1080},
		{"https://v.redd.it/cmaf_720.mp4", P720},
		{"https://example.com/movie_4k.mp4", P4K},
		{"https://example.com/clip_2k.mp4", P2K},
		{"video_540p_stream", P540},
		{"download_480.mp4", P480},
		{"sample_360.mp4", P360},
		{"sample_240.mp4", P240},
		{"sample_144.mp4", P144},
		{"generic_video_url", HD},
	}

	for _, tt := range tests {
		got := Parse(tt.input)
		if got != tt.expected {
			t.Errorf("Parse(%q) = %s, expected %s", tt.input, got, tt.expected)
		}
	}
}

func TestFormatters(t *testing.T) {
	if got := FromDimensions(1920, 1080); got != "1920x1080" {
		t.Errorf("FromDimensions(1920, 1080) = %s, expected 1920x1080", got)
	}

	if got := FromHeight(720); got != "720p" {
		t.Errorf("FromHeight(720) = %s, expected 720p", got)
	}
}

func TestSortItems(t *testing.T) {
	items := []media.Item{
		{Label: "Low", Quality: "360p"},
		{Label: "Top", Quality: "original"},
		{Label: "HD", Quality: "1080p"},
		{Label: "Mid", Quality: "720p"},
	}

	SortItems(items)

	expectedOrder := []string{"original", "1080p", "720p", "360p"}
	for i, it := range items {
		if it.Quality != expectedOrder[i] {
			t.Errorf("item[%d].Quality = %s, expected %s", i, it.Quality, expectedOrder[i])
		}
	}
}

type testVideoVersion struct {
	Width  int
	Height int
	Name   string
}

func TestBestDimension(t *testing.T) {
	versions := []testVideoVersion{
		{Width: 720, Height: 1280, Name: "720p"},
		{Width: 1080, Height: 1920, Name: "1080p"},
		{Width: 360, Height: 640, Name: "360p"},
	}

	best := BestDimension(versions, func(v testVideoVersion) (int, int) {
		return v.Width, v.Height
	})

	if best == nil || best.Name != "1080p" {
		t.Fatalf("expected 1080p, got %+v", best)
	}

	// Empty slice returns nil
	var empty []testVideoVersion
	if got := BestDimension(empty, func(v testVideoVersion) (int, int) { return v.Width, v.Height }); got != nil {
		t.Errorf("expected nil for empty slice, got %+v", got)
	}
}

func TestPinterestPresets(t *testing.T) {
	presets := PinterestPresets()
	if len(presets) == 0 {
		t.Fatal("expected non-empty presets")
	}

	// Ensure modifications don't mutate base StandardPresets
	presets[0].Label = "modified"
	if StandardPresets[0].Label == "modified" {
		t.Error("PinterestPresets should return an isolated copy")
	}
}
