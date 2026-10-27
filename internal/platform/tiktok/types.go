package tiktok

import "github.com/pavelc4/astra/internal/media"

type UserInfo struct {
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

type Result struct {
	Platform  string       `json:"platform"`
	Title     *string      `json:"title"`
	Thumbnail *string      `json:"thumbnail"`
	User      *UserInfo    `json:"user,omitempty"`
	Downloads []media.Item `json:"downloads"`
}

type UserResult struct {
	Platform     string `json:"platform"`
	Username     string `json:"username"`
	Nickname     string `json:"nickname"`
	AvatarThumb  string `json:"avatar_thumb"`
	Avatar       string `json:"avatar"`
	AvatarLarger string `json:"avatar_larger"`
	Signature    string `json:"signature"`
	Verified     bool   `json:"verified"`
	Followers    int    `json:"followers"`
	Following    int    `json:"following"`
	Likes        int    `json:"likes"`
	Videos       int    `json:"videos"`
}
