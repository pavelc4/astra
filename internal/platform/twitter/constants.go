package twitter

import "regexp"

const (
	PlatformName        = "twitter"
	SaveTwitterEndpoint = "https://savetwitter.net/api/ajaxSearch"
	RefererURL          = "https://savetwitter.net/en4"
	OriginURL           = "https://savetwitter.net"
)

var (
	QualityRegex = regexp.MustCompile(`\((\d+p)\)`)
	URLRegex     = regexp.MustCompile(`https?://(?:www\.)?(?:twitter\.com|x\.com)/`)
)
