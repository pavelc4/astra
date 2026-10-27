package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strings"
)

// NormalizeURL canonicalizes a social media URL by trimming whitespace,
// lowercasing the scheme and host, stripping 'www.' and mobile 'm.' subdomains,
// aliasing known domains (e.g. x.com -> twitter.com), stripping tracking parameters,
// and removing trailing slashes.
func NormalizeURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	if parsed.Scheme == "" || parsed.Scheme == "http" {
		parsed.Scheme = "https"
	} else {
		parsed.Scheme = strings.ToLower(parsed.Scheme)
	}

	host := strings.ToLower(parsed.Host)
	host = strings.TrimPrefix(host, "www.")
	host = strings.TrimPrefix(host, "m.")

	switch host {
	case "x.com":
		host = "twitter.com"
	}
	parsed.Host = host

	q := parsed.Query()
	trackingParams := []string{
		"utm_source", "utm_medium", "utm_campaign", "utm_term", "utm_content",
		"igsh", "igshid", "fbclid", "gclid", "share_item", "share_link_id",
		"share_app_id", "share_id", "social_sharing", "t", "s", "ref",
		"ref_src", "ref_url", "source", "feature", "si", "is_from_webapp",
		"sender_device", "tt_from", "context", "entry_point", "app",
		"mibextid", "rdid",
	}

	for _, param := range trackingParams {
		q.Del(param)
	}

	parsed.RawQuery = q.Encode()
	parsed.Fragment = ""

	// Strip trailing slash for paths with > 1 segment (e.g. /reel/123/ -> /reel/123)
	if len(parsed.Path) > 1 && strings.HasSuffix(parsed.Path, "/") {
		parsed.Path = strings.TrimSuffix(parsed.Path, "/")
	}

	return parsed.String()
}

// HashKey produces a deterministic 64-character SHA-256 hex string from an operation and clean URL.
func HashKey(operation, cleanURL string) string {
	sum := sha256.Sum256([]byte(operation + "|" + cleanURL))
	return hex.EncodeToString(sum[:])
}
