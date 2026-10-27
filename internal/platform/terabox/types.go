package terabox

import "encoding/json"

type teraboxFile struct {
	Name         string          `json:"name"`
	Size         string          `json:"size"`
	FileType     string          `json:"file_type"`
	DownloadURL  string          `json:"download_url"`
	ZipURL       string          `json:"zip_url"`
	ThumbnailURL string          `json:"thumbnail_url"`
	StreamURLs   json.RawMessage `json:"stream_urls"`
	SubtitleURL  string          `json:"subtitle_url"`
	Duration     string          `json:"duration"`
	Quality      string          `json:"quality"`
}

type teraboxData struct {
	Status string        `json:"status"`
	Files  []teraboxFile `json:"files"`
}
