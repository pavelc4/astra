package twitter

import "github.com/pavelc4/astra/internal/media"

type Result struct {
	Platform  string       `json:"platform"`
	Type      string       `json:"type"`
	TweetID   *string      `json:"tweetId"`
	Title     *string      `json:"title"`
	Duration  *string      `json:"duration"`
	Thumbnail *string      `json:"thumbnail"`
	Downloads []media.Item `json:"downloads"`
}

type AjaxResponse struct {
	Status string `json:"status"`
	Data   string `json:"data"`
}
