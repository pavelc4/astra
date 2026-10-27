package tiktok

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/media"
)

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

	var downloads []media.Item

	if post.Play != "" {
		downloads = append(downloads, media.Item{Label: "No Watermark", URL: post.Play, Type: media.Video})
	}
	if post.Hdplay != "" {
		downloads = append(downloads, media.Item{Label: "HD", URL: post.Hdplay, Type: media.Video})
	}
	if post.Original != "" {
		label := "Original"
		if post.OriginalSize > 0 {
			label = fmt.Sprintf("Original (%dMB)", post.OriginalSize/1024/1024)
		}
		downloads = append(downloads, media.Item{Label: label, URL: post.Original, Type: media.Video})
	}
	if post.Wmplay != "" {
		downloads = append(downloads, media.Item{Label: "With Watermark", URL: post.Wmplay, Type: media.Video})
	}

	if post.Music != "" {
		downloads = append(downloads, media.Item{Label: "Audio", URL: post.Music, Type: media.Audio})
	}
	for _, img := range post.Images {
		if img != "" {
			downloads = append(downloads, media.Item{URL: img, Type: media.Image})
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
