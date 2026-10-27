package reddit

import "regexp"

const (
	PlatformName = "reddit"
	RapidSaveURL = "https://rapidsave.com/info"
	RefererURL   = "https://rapidsave.com/"
)

var (
	VRedditRegex = regexp.MustCompile(`https://v\.redd\.it/[^\s"<&]+`)
	ImgReddRegex = regexp.MustCompile(`https://i\.redd\.it/[^\s"<]+`)
	URLRegex     = regexp.MustCompile(`https?://(?:(?:www\.|old\.|new\.)?reddit\.com/|(?:v|i)\.redd\.it/)`)
)
