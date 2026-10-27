package soundcloud

import "regexp"

const (
	PlatformName = "soundcloud"
	APIEndpoint  = "https://urlmp4.com/wp-json/aio-dl/video-data/"
	RefererURL   = "https://urlmp4.com/en/soundcloud-downloader/"
	DefaultToken = "8b6e170975d92939bb67d8db567f82e43fa2da91e00a84f258af77c1186c5e8a"
	HashSalt     = "1043YWlvLWRs"
)

var (
	URLRegex = regexp.MustCompile(`https?://(?:on\.)?soundcloud\.com/`)
)
