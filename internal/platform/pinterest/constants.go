package pinterest

import "regexp"

const (
	PlatformName = "pinterest"
)

var (
	URLRegex      = regexp.MustCompile(`https?://(?:[a-zA-Z0-9-]+\.)?(?:pinterest\.[a-z.]+|pin\.it)/`)
	PinIDRegex    = regexp.MustCompile(`/pin/(\d+)`)
	M3U8SigRegex  = regexp.MustCompile(`/hls/([0-9a-f]{2})/([0-9a-f]{2})/([0-9a-f]{2})/([0-9a-f]{32})\.m3u8`)
	StreamEntryRe = regexp.MustCompile(`RESOLUTION=(\d+x\d+)[^\n]*\n([^\n]+_([^.\n]+)\.m3u8)`)
)
