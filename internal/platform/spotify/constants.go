package spotify

import "regexp"

const (
	PlatformName     = "spotify"
	MusicFabEndpoint = "https://musicfab.io/api/spotify"
	RefererURL       = "https://musicfab.io/"
	OriginURL        = "https://musicfab.io"
)

var (
	URLRegex = regexp.MustCompile(`https?://(?:open\.)?spotify\.com/`)
)
