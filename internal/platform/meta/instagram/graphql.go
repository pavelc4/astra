package instagram

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pavelc4/astra/internal/resolution"
)

func (c *IGClient) FetchProfile(ctx context.Context, username string) (*UserProfile, error) {
	url := fmt.Sprintf("%s/users/%s/usernameinfo/", apiURL, username)
	data, err := c.getJSON(ctx, url)
	if err != nil {
		return nil, err
	}

	var resp struct {
		User struct {
			Pk              string `json:"pk"`
			Username        string `json:"username"`
			FullName        string `json:"full_name"`
			ProfilePicURL   string `json:"profile_pic_url"`
			ProfilePicURLHD string `json:"profile_pic_url_hd"`
			Biography       string `json:"biography"`
			FollowerCount   int    `json:"follower_count"`
			FollowingCount  int    `json:"following_count"`
			MediaCount      int    `json:"media_count"`
			IsVerified      bool   `json:"is_verified"`
			IsPrivate       bool   `json:"is_private"`
			ExternalURL     string `json:"external_url"`
		} `json:"user"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse profile: %w", err)
	}

	if resp.User.Username == "" {
		return nil, fmt.Errorf("user not found")
	}

	return &UserProfile{
		Username:    resp.User.Username,
		FullName:    resp.User.FullName,
		Avatar:      resp.User.ProfilePicURL,
		AvatarHD:    resp.User.ProfilePicURLHD,
		Biography:   resp.User.Biography,
		Followers:   resp.User.FollowerCount,
		Following:   resp.User.FollowingCount,
		Posts:       resp.User.MediaCount,
		Verified:    resp.User.IsVerified,
		Private:     resp.User.IsPrivate,
		ExternalURL: resp.User.ExternalURL,
	}, nil
}

func (c *IGClient) FetchMedia(ctx context.Context, longURL string) (*MediaInfo, error) {
	shortcode := extractShortcode(longURL)
	if shortcode == "" {
		return nil, fmt.Errorf("could not extract shortcode from URL")
	}

	mediaID, err := extractMediaID(ctx, shortcode)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/media/%s/info/", apiURL, mediaID)
	data, err := c.getJSON(ctx, url)
	if err != nil {
		return nil, err
	}

	var resp mobileInfoResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse media: %w", err)
	}

	if len(resp.Items) == 0 {
		return nil, fmt.Errorf("media not found")
	}

	return convertMediaItem(&resp.Items[0]), nil
}

// getUserPK fetches the numeric user ID for a given Instagram username.
func (c *IGClient) getUserPK(ctx context.Context, username string) (string, error) {
	url := fmt.Sprintf("%s/users/%s/usernameinfo/", apiURL, username)
	data, err := c.getJSON(ctx, url)
	if err != nil {
		return "", err
	}

	var resp struct {
		User struct {
			Pk string `json:"pk"`
		} `json:"user"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("parse user pk: %w", err)
	}
	if resp.User.Pk == "" {
		return "", fmt.Errorf("user not found")
	}
	return resp.User.Pk, nil
}

func (c *IGClient) FetchStories(ctx context.Context, username string) (*StoriesResult, error) {
	pk, err := c.getUserPK(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("get user id: %w", err)
	}

	url := fmt.Sprintf("%s/feed/user/%s/reel_media/", apiURL, pk)
	data, err := c.getJSON(ctx, url)
	if err != nil {
		return nil, err
	}

	var resp struct {
		ID    string `json:"id"`
		Items []struct {
			Pk             string  `json:"pk"`
			MediaType      int     `json:"media_type"`
			TakenAt        int64   `json:"taken_at"`
			VideoDuration  float64 `json:"video_duration,omitempty"`
			OriginalWidth  int     `json:"original_width"`
			OriginalHeight int     `json:"original_height"`
			VideoVersions  []struct {
				Type int    `json:"type"`
				URL  string `json:"url"`
			} `json:"video_versions"`
			ImageVersions2 *struct {
				Candidates []struct {
					URL    string `json:"url"`
					Width  int    `json:"width"`
					Height int    `json:"height"`
				} `json:"candidates"`
			} `json:"image_versions2"`
		} `json:"items"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse stories: %w", err)
	}

	result := &StoriesResult{Username: username}
	for _, item := range resp.Items {
		story := StoryItem{TakenAt: item.TakenAt}

		if item.MediaType == 2 {
			for _, v := range item.VideoVersions {
				story.Videos = append(story.Videos, MediaItem{URL: v.URL})
			}
		} else {
			if item.ImageVersions2 != nil && len(item.ImageVersions2.Candidates) > 0 {
				story.Images = append(story.Images, MediaItem{URL: item.ImageVersions2.Candidates[0].URL})
			}
		}

		result.Items = append(result.Items, story)
	}

	if len(result.Items) == 0 {
		return nil, fmt.Errorf("no stories available")
	}

	return result, nil
}

func convertMediaItem(item *mobileMediaItem) *MediaInfo {
	result := &MediaInfo{}

	if item.Caption != nil {
		result.Caption = item.Caption.Text
	}
	if item.User != nil {
		result.OwnerUser = item.User.Username
	}

	switch item.MediaType {
	case 2: // single video
		if v := bestVideoItem(item.VideoVersions); v != nil {
			result.Videos = append(result.Videos, *v)
		}
	case 8: // carousel
		for _, cm := range item.CarouselMedia {
			if cm.MediaType == 2 {
				if v := bestVideoItem(cm.VideoVersions); v != nil {
					result.Videos = append(result.Videos, *v)
				}
			} else if p := firstPhotoItem(cm.ImageVersions2); p != nil {
				result.Photos = append(result.Photos, *p)
			}
		}
	default: // single photo
		if p := firstPhotoItem(item.ImageVersions2); p != nil {
			result.Photos = append(result.Photos, *p)
		}
	}

	// audio from clips metadata
	if item.ClipsMetadata != nil && item.ClipsMetadata.MusicInfo != nil {
		result.AudioURL = item.ClipsMetadata.MusicInfo.AudioSrc
	}

	result.Items = append(result.Photos, result.Videos...)

	return result
}

func bestVideoItem(versions []mobileVideoVersion) *MediaItem {
	best := resolution.BestDimension(versions, func(v mobileVideoVersion) (int, int) {
		return v.Width, v.Height
	})
	if best == nil {
		return nil
	}
	return &MediaItem{
		Quality: resolution.FromDimensions(best.Width, best.Height),
		URL:     best.URL,
	}
}

func firstPhotoItem(img *mobileImageVersions) *MediaItem {
	if img != nil && len(img.Candidates) > 0 {
		return &MediaItem{URL: img.Candidates[0].URL}
	}
	return nil
}
