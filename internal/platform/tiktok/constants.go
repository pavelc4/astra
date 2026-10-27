package tiktok

import (
	"regexp"
	"time"
)

const (
	PlatformName   = "tiktok"
	BaseAPIURL     = "https://tikwm.com/api"
	DefaultTimeout = time.Second + 100*time.Millisecond
)

var (
	// URLRegex matches standard, mobile, and shortlink TikTok URLs
	URLRegex = regexp.MustCompile(`https?://(?:[a-zA-Z0-9-]+\.)?tiktok\.com/`)
)
