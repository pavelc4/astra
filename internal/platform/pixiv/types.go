package pixiv

// Artwork is the summary returned by the profile/illustrations/bookmarks list
// endpoints. Thumb may be empty without a (valid) cookie; pageUrl always works.
type Artwork struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Thumb   string `json:"thumb,omitempty"`
	PageURL string `json:"pageUrl"`
}

// illustBody is the subset of the /ajax/illust/{id} response we need.
type illustBody struct {
	IllustType int    `json:"illustType"` // 0=illust, 1=manga, 2=ugoira
	Title      string `json:"title"`
	PageCount  int    `json:"pageCount"`
	UserName   string `json:"userName"`
	URLs       struct {
		Original string `json:"original"`
		Regular  string `json:"regular"`
	} `json:"urls"`
}

type illustResp struct {
	Error   bool        `json:"error"`
	Message string      `json:"message"`
	Body    *illustBody `json:"body"`
}

// worksResp is the {works:[...]} part shared by the illusts/tag and bookmarks lists.
type worksResp struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
	Body    struct {
		Works []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
			URLs  struct {
				Regular string `json:"regular"`
			} `json:"urls"`
		} `json:"works"`
	} `json:"body"`
}
