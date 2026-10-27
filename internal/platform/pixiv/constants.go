package pixiv

import "regexp"

const (
	baseURL = "https://www.pixiv.net"
	referer = "https://www.pixiv.net/"
)

var (
	reArtworkID = regexp.MustCompile(`/artworks/(\d+)/?$`)
	reUserID    = regexp.MustCompile(`/users/(\d+)/?`)
)

// Supported ranking modes (from PixivBatchDownloader)
var validRankingModes = map[string]bool{
	"daily":       true,
	"weekly":      true,
	"monthly":     true,
	"rookie":      true,
	"original":    true,
	"male":        true,
	"female":      true,
	"daily_r18":   true,
	"weekly_r18":  true,
	"male_r18":    true,
	"female_r18":  true,
	"r18g":        true,
}
