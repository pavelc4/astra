package soundcloud

type RawMedia struct {
	URL            string `json:"url"`
	Quality        string `json:"quality"`
	Extension      string `json:"extension"`
	Size           int64  `json:"size"`
	AudioAvailable bool   `json:"audioAvailable"`
}

type RawResponse struct {
	Title     string     `json:"title"`
	Thumbnail string     `json:"thumbnail"`
	Medias    []RawMedia `json:"medias"`
}
