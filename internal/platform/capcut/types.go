package capcut

import "encoding/json"

type Result struct {
	Platform string          `json:"platform"`
	Raw      json.RawMessage `json:"raw"`
}
