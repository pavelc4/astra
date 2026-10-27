package resolution

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/pavelc4/astra/internal/media"
)

var (
	reResolutionP = regexp.MustCompile(`(?i)(\d{3,4})p`)
	reDimension   = regexp.MustCompile(`(?i)(\d{3,4})\s*[x×]\s*(\d{3,4})`)
)

// Map-based lookup table for predefined quality rankings (O(1) dictionary match).
var qualityRanks = map[string]int{
	"4k":       2160,
	"2k":       1440,
	"1080p":    1080,
	"hd":       1080,
	"720p":     720,
	"540p":     540,
	"480p":     480,
	"sd":       480,
	"360p":     360,
	"240p":     240,
	"144p":     144,
	"original": 9999,
}

// Table-driven match rules for URL / filename pattern matching (mimicking Rust match arms).
var resolutionRules = [...]struct {
	pattern string
	label   string
}{
	{"2160", P4K},
	{"4k", P4K},
	{"1440", P2K},
	{"2k", P2K},
	{"1080", P1080},
	{"720", P720},
	{"540", P540},
	{"480", P480},
	{"360", P360},
	{"240", P240},
	{"144", P144},
}

// Rank returns a numerical weight for comparing resolution labels.
// Replaces switch-case with Table Lookup & Regex extractors.
func Rank(quality string) int {
	q := strings.ToLower(strings.TrimSpace(quality))
	if q == "" {
		return 0
	}

	// 1. O(1) Dictionary match
	if rank, ok := qualityRanks[q]; ok {
		return rank
	}

	// 2. Extract height from patterns like "1080p"
	if m := reResolutionP.FindStringSubmatch(q); len(m) > 1 {
		if val, err := strconv.Atoi(m[1]); err == nil {
			return val
		}
	}

	// 3. Extract dimensions like "1920x1080"
	if m := reDimension.FindStringSubmatch(q); len(m) > 2 {
		w, _ := strconv.Atoi(m[1])
		h, _ := strconv.Atoi(m[2])
		if w > 0 && h > 0 {
			if w < h {
				return w
			}
			return h
		}
	}

	// 4. Extract width-suffix like "720w"
	if val, err := strconv.Atoi(strings.TrimSuffix(q, "w")); err == nil {
		return val
	}

	return 0
}

// Parse extracts a standard quality label from a URL, filename, or text string
// using a Table-Driven Match rule set (zero switch-case).
func Parse(raw string) string {
	lower := strings.ToLower(raw)

	for _, rule := range resolutionRules {
		if strings.Contains(lower, rule.pattern) {
			return rule.label
		}
	}

	return HD
}

// FromDimensions formats width and height into a standard "WIDTHxHEIGHT" string.
func FromDimensions(width, height int) string {
	return fmt.Sprintf("%dx%d", width, height)
}

// FromHeight formats vertical pixel count into a standard label (e.g. 1080 -> "1080p").
func FromHeight(height int) string {
	return fmt.Sprintf("%dp", height)
}

// SortItems sorts a slice of media items in-place descending by quality resolution.
func SortItems(items []media.Item) {
	sort.SliceStable(items, func(i, j int) bool {
		return Rank(items[i].Quality) > Rank(items[j].Quality)
	})
}

// BestDimension finds the element with the highest total pixel count (width * height).
func BestDimension[T any](items []T, dimFn func(T) (int, int)) *T {
	if len(items) == 0 {
		return nil
	}

	bestIdx := 0
	w0, h0 := dimFn(items[0])
	bestPixels := w0 * h0

	for i := 1; i < len(items); i++ {
		w, h := dimFn(items[i])
		if pixels := w * h; pixels > bestPixels {
			bestPixels = pixels
			bestIdx = i
		}
	}

	return &items[bestIdx]
}

// PinterestPresets returns pre-defined Pinterest quality presets.
// Avoids continuous HTTP GET calls to master m3u8 on Pinterest video downloads.
func PinterestPresets() []QualityPreset {
	out := make([]QualityPreset, len(StandardPresets))
	copy(out, StandardPresets)
	return out
}
