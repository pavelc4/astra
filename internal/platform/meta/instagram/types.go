package instagram

import (
	"regexp"
	"strings"
)



type MediaItem struct {
	Quality   string  `json:"quality"`
	Thumbnail *string `json:"thumbnail,omitempty"`
	URL       string  `json:"url"`
}

type UserProfile struct {
	Username    string `json:"username"`
	FullName    string `json:"full_name"`
	Avatar      string `json:"avatar"`
	AvatarHD    string `json:"avatar_hd"`
	Biography   string `json:"biography"`
	Followers   int    `json:"followers"`
	Following   int    `json:"following"`
	Posts       int    `json:"posts"`
	Verified    bool   `json:"verified"`
	Private     bool   `json:"private"`
	ExternalURL string `json:"external_url,omitempty"`
}

type StoryItem struct {
	TakenAt int64       `json:"taken_at"`
	Images  []MediaItem `json:"images,omitempty"`
	Videos  []MediaItem `json:"videos,omitempty"`
}

type StoriesResult struct {
	Username string      `json:"username"`
	Items    []StoryItem `json:"items"`
}

type MediaInfo struct {
	Items       []MediaItem `json:"items"`
	Caption     string      `json:"caption,omitempty"`
	OwnerAvatar string      `json:"owner_avatar,omitempty"`
	OwnerUser   string      `json:"owner_username,omitempty"`
	AudioURL    string      `json:"audio_url,omitempty"`
	Photos      []MediaItem `json:"photos,omitempty"`
	Videos      []MediaItem `json:"videos,omitempty"`
}

func ExtractUsername(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	rawURL = strings.TrimSuffix(rawURL, "/")

	if idx := strings.Index(rawURL, "instagram.com/"); idx != -1 {
		path := rawURL[idx+14:]

		if strings.HasPrefix(path, "p/") || strings.HasPrefix(path, "reel/") || strings.HasPrefix(path, "reels/") || strings.HasPrefix(path, "tv/") {
			return ""
		}

		if e := strings.Index(path, "/"); e != -1 {
			path = path[:e]
		}
		if e := strings.Index(path, "?"); e != -1 {
			path = path[:e]
		}
		if path != "" && !strings.ContainsAny(path, "?#") {
			return path
		}
	}

	return ""
}

var igPathRe = regexp.MustCompile(`/(?:p|reel|reels|tv|share/(?:p|reel|reels))/([A-Za-z0-9_-]+)`)

func extractShortcode(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if m := igPathRe.FindStringSubmatch(rawURL); len(m) > 1 {
		return m[1]
	}
	return ""
}

