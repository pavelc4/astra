package spotify

type SpotifyMetadata struct {
	Name     string `json:"name"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	Duration string `json:"duration"`
	Image    string `json:"image"`
	Download string `json:"download"`
}

type SpotifyData struct {
	Metadata SpotifyMetadata `json:"metadata"`
}

type SpotifyRawResponse struct {
	Data SpotifyData `json:"data"`
}
