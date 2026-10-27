package resolution

// Standard quality label definitions.
const (
	P4K   = "4K"
	P2K   = "2K"
	P1080 = "1080p"
	P720  = "720p"
	P540  = "540p"
	P480  = "480p"
	P360  = "360p"
	P240  = "240p"
	P144  = "144p"

	HD       = "hd"
	SD       = "sd"
	Original = "original"
	HLS      = "hls"
	Gallery  = "gallery"
	Ugoira   = "ugoira"
)

// QualityPreset defines a named resolution preset with its suffix, label, and vertical pixel height.
type QualityPreset struct {
	Suffix string `json:"suffix"`
	Label  string `json:"label"`
	Height int    `json:"height"`
}

// StandardPresets provides descending ordered standard presets.
var StandardPresets = []QualityPreset{
	{Suffix: "1080w", Label: P1080, Height: 1080},
	{Suffix: "720w", Label: P720, Height: 720},
	{Suffix: "540w", Label: P540, Height: 540},
	{Suffix: "360w", Label: P360, Height: 360},
	{Suffix: "240w", Label: P240, Height: 240},
}
