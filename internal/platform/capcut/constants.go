package capcut

import "regexp"

const (
	PlatformName     = "capcut"
	GenViralEndpoint = "https://www.genviral.io/api/tools/social-downloader"
	OriginURL        = "https://www.genviral.io"
	RefererURL       = "https://www.genviral.io/tools/download/capcut"
)

var (
	URLRegex = regexp.MustCompile(`https?://(?:www\.)?capcut\.com/`)
)
