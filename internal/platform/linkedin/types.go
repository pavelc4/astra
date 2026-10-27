package linkedin

import "github.com/pavelc4/astra/internal/media"

type Result struct {
	Platform  string       `json:"platform"`
	Title     string       `json:"title"`
	Thumbnail string       `json:"thumbnail,omitempty"`
	Downloads []media.Item `json:"downloads"`
}

type SayWhatVideo struct {
	URL     string `json:"url"`
	Quality string `json:"quality"`
}

type SayWhatResponse struct {
	Videos []SayWhatVideo `json:"videos"`
}
