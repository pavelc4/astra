package tiktok

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/types"
)

type UserInfo struct {
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

type Result struct {
	Platform  string               `json:"platform"`
	Title     *string              `json:"title"`
	Thumbnail *string              `json:"thumbnail"`
	User      *UserInfo            `json:"user,omitempty"`
	Downloads []types.DownloadItem `json:"downloads"`
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

func FetchData(ctx context.Context, videoURL string) (*Result, error) {
	post, err := GetPostOriginal(ctx, videoURL)
	if err != nil {
		return nil, errors.NewUpstream(fmt.Sprintf("TikTok fetch failed: %s", err.Error()))
	}
	if post == nil {
		return nil, errors.NewUpstream("TikTok returned empty response")
	}

	var title *string
	if post.Title != "" {
		title = &post.Title
	}
	var thumbnail *string
	if post.Cover != "" {
		thumbnail = &post.Cover
	}

	var downloads []types.DownloadItem

	if post.Play != "" {
		downloads = append(downloads, types.DownloadItem{Label: "No Watermark", URL: post.Play, Type: types.MediaVideo})
	}
	if post.Hdplay != "" {
		downloads = append(downloads, types.DownloadItem{Label: "HD", URL: post.Hdplay, Type: types.MediaVideo})
	}
	if post.Original != "" {
		label := "Original"
		if post.OriginalSize > 0 {
			label = fmt.Sprintf("Original (%dMB)", post.OriginalSize/1024/1024)
		}
		downloads = append(downloads, types.DownloadItem{Label: label, URL: post.Original, Type: types.MediaVideo})
	}
	if post.Wmplay != "" {
		downloads = append(downloads, types.DownloadItem{Label: "With Watermark", URL: post.Wmplay, Type: types.MediaVideo})
	}

	if post.Music != "" {
		downloads = append(downloads, types.DownloadItem{Label: "Audio", URL: post.Music, Type: types.MediaAudio})
	}
	for _, img := range post.Images {
		if img != "" {
			downloads = append(downloads, types.DownloadItem{URL: img, Type: types.MediaImage})
		}
	}

	result := &Result{
		Platform:  "tiktok",
		Title:     title,
		Thumbnail: thumbnail,
		Downloads: downloads,
	}
	if post.Author.UniqueId != "" {
		result.User = &UserInfo{
			Username: post.Author.UniqueId,
			Nickname: post.Author.Nickname,
			Avatar:   post.Author.Avatar,
		}
	}
	return result, nil
}

func FetchUser(ctx context.Context, profileURL string) (*UserResult, error) {
	username := extractUsername(profileURL)
	if username == "" {
		return nil, errors.NewInvalidURL("could not extract username from URL")
	}

	detail, err := GetUserDetail(ctx, username)
	if err != nil {
		return nil, errors.NewUpstream(fmt.Sprintf("TikTok user fetch failed: %s", err.Error()))
	}
	if detail == nil {
		return nil, errors.NewUpstream("TikTok returned empty user response")
	}

	return &UserResult{
		Platform:     "tiktok",
		Username:     detail.User.UniqueId,
		Nickname:     detail.User.Nickname,
		AvatarThumb:  detail.User.AvatarThumb,
		Avatar:       detail.User.AvatarMedium,
		AvatarLarger: detail.User.AvatarLarger,
		Signature:    detail.User.Signature,
		Verified:     detail.User.Verified,
		Followers:    detail.Stats.FollowerCount,
		Following:    detail.Stats.FollowingCount,
		Likes:        detail.Stats.HeartCount,
		Videos:       detail.Stats.VideoCount,
	}, nil
}

type MusicResult struct {
	Platform string `json:"platform"`
	ID       string `json:"id"`
	Title    string `json:"title"`
	Audio    string `json:"audio"`
	Cover    string `json:"cover"`
	Author   string `json:"author"`
	Duration int    `json:"duration"`
	Videos   int    `json:"videos"`
}

func FetchMusic(ctx context.Context, musicURL string) (*MusicResult, error) {
	detail, err := GetMusicDetail(ctx, musicURL)
	if err != nil {
		return nil, errors.NewUpstream(fmt.Sprintf("TikTok music fetch failed: %s", err.Error()))
	}
	if detail == nil {
		return nil, errors.NewUpstream("TikTok returned empty music response")
	}

	return &MusicResult{
		Platform: "tiktok",
		ID:       detail.Id,
		Title:    detail.Title,
		Audio:    detail.Play,
		Cover:    detail.Cover,
		Author:   detail.Author,
		Duration: detail.Duration,
		Videos:   detail.VideoCount,
	}, nil
}

func extractUsername(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	// Parse as URL if it looks like one
	if strings.HasPrefix(s, "http") || strings.HasPrefix(s, "www.") {
		if !strings.HasPrefix(s, "http") {
			s = "https://" + s
		}
		u, err := url.Parse(s)
		if err != nil {
			return ""
		}
		s = u.Path
	}

	s = strings.TrimPrefix(s, "/")
	s = strings.TrimPrefix(s, "@")
	return s
}
