package engine

import (
	"context"
)

// Extractor defines the unified interface for extracting media from a platform URL.
type Extractor interface {
	// Name returns the identifier of the platform (e.g. "tiktok", "instagram").
	Name() string

	// Matches returns true if the extractor can handle the given raw URL.
	Matches(rawURL string) bool

	// Extract fetches media metadata and download URLs for the target URL.
	Extract(ctx context.Context, rawURL string) (any, error)
}
