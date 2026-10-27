package linkedin

import "regexp"

const (
	PlatformName    = "linkedin"
	SayWhatEndpoint = "https://saywhat.ai/api/fetch-linkedin-page/"
	SayWhatReferer  = "https://saywhat.ai/tools/linkedin-video-downloader/"
	GooglebotUA     = "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"
)

var (
	URLRegex = regexp.MustCompile(`https?://(?:www\.)?linkedin\.com/`)
)
