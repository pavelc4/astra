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
