package pixiv

// Artwork is the summary returned by the profile/illustrations/bookmarks list
// endpoints. Thumb may be empty without a (valid) cookie; pageUrl always works.
type Artwork struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Thumb   string `json:"thumb,omitempty"`
	PageURL string `json:"pageUrl"`
}

// UgoiraFrame represents a single animation frame and duration in milliseconds.
type UgoiraFrame struct {
	File  string `json:"file"`
	Delay int    `json:"delay"`
}

// UgoiraMeta contains animated illustration metadata from /ajax/illust/{id}/ugoira_meta.
type UgoiraMeta struct {
	Src         string        `json:"src"`
	OriginalSrc string        `json:"originalSrc"`
	MimeType    string        `json:"mime_type"`
	Frames      []UgoiraFrame `json:"frames"`
}

type ugoiraResp struct {
	Error   bool        `json:"error"`
	Message string      `json:"message"`
	Body    *UgoiraMeta `json:"body"`
}

// RankingItem represents an artwork entry in ranking.php.
type RankingItem struct {
	Rank        int      `json:"rank"`
	IllustID    int64    `json:"illust_id"`
	Title       string   `json:"title"`
	UserID      int64    `json:"user_id"`
	UserName    string   `json:"user_name"`
	URL         string   `json:"url"`
	PageURL     string   `json:"pageUrl"`
	PageCount   int      `json:"illust_page_count"`
	ViewCount   int      `json:"view_count"`
	RatingCount int      `json:"rating_count"`
	Tags        []string `json:"tags"`
}

type rankingResp struct {
	Contents []RankingItem `json:"contents"`
	Mode     string        `json:"mode"`
	Date     string        `json:"date"`
	Page     int           `json:"page"`
}

// UserAllWorks maps all artwork/novel IDs created by a user from /ajax/user/{id}/profile/all.
type UserAllWorks struct {
	Illusts []string `json:"illusts"`
	Manga   []string `json:"manga"`
	Novels  []string `json:"novels"`
}

type userProfileAllResp struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
	Body    struct {
		Illusts map[string]any `json:"illusts"`
		Manga   map[string]any `json:"manga"`
		Novels  map[string]any `json:"novels"`
	} `json:"body"`
}

// illustBody is the full subset of the /ajax/illust/{id} response.
type illustBody struct {
	IllustType    int    `json:"illustType"` // 0=illust, 1=manga, 2=ugoira
	Title         string `json:"title"`
	Description   string `json:"description"`
	PageCount     int    `json:"pageCount"`
	UserName      string `json:"userName"`
	UserID        string `json:"userId"`
	AIType        int    `json:"aiType"`    // 1=human, 2=ai
	XRestrict     int    `json:"xRestrict"` // 0=all, 1=r18, 2=r18g
	BookmarkCount int    `json:"bookmarkCount"`
	LikeCount     int    `json:"likeCount"`
	ViewCount     int    `json:"viewCount"`
	UploadDate    string `json:"uploadDate"`
	Tags          struct {
		Tags []struct {
			Tag         string `json:"tag"`
			Translation *struct {
				EN string `json:"en"`
			} `json:"translation,omitempty"`
		} `json:"tags"`
	} `json:"tags"`
	URLs struct {
		Original string `json:"original"`
		Regular  string `json:"regular"`
		Small    string `json:"small"`
		Thumb    string `json:"thumb"`
		Mini     string `json:"mini"`
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
