package engine

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/platform/capcut"
	"github.com/pavelc4/astra/internal/platform/linkedin"
	"github.com/pavelc4/astra/internal/platform/meta/facebook"
	"github.com/pavelc4/astra/internal/platform/meta/instagram"
	"github.com/pavelc4/astra/internal/platform/meta/threads"
	"github.com/pavelc4/astra/internal/platform/pinterest"
	"github.com/pavelc4/astra/internal/platform/pixiv"
	"github.com/pavelc4/astra/internal/platform/reddit"
	"github.com/pavelc4/astra/internal/platform/soundcloud"
	"github.com/pavelc4/astra/internal/platform/spotify"
	"github.com/pavelc4/astra/internal/platform/terabox"
	"github.com/pavelc4/astra/internal/platform/tiktok"
	"github.com/pavelc4/astra/internal/platform/twitter"
)

// Registry manages the set of available media extractors.
type Registry struct {
	mu         sync.RWMutex
	extractors []Extractor
}

// NewRegistry initializes an empty extractor registry.
func NewRegistry() *Registry {
	return &Registry{
		extractors: make([]Extractor, 0),
	}
}

// Register appends an extractor to the registry.
func (r *Registry) Register(e Extractor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.extractors = append(r.extractors, e)
}

// Resolve finds the first extractor that can handle rawURL.
// If no extractor matches, returns an INVALID_URL AppError.
func (r *Registry) Resolve(rawURL string) (Extractor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, e := range r.extractors {
		if e.Matches(rawURL) {
			return e, nil
		}
	}
	return nil, errors.NewInvalidURL(fmt.Sprintf("no extractor found for URL: %s", rawURL))
}

// Extractors returns a snapshot copy of all registered extractors.
func (r *Registry) Extractors() []Extractor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Extractor, len(r.extractors))
	copy(out, r.extractors)
	return out
}

type genericExtractor struct {
	name    string
	matcher func(host string) bool
	fetch   func(ctx context.Context, rawURL string) (any, error)
}

func (g *genericExtractor) Name() string { return g.name }
func (g *genericExtractor) Matches(rawURL string) bool {
	return matchHost(rawURL, g.matcher)
}
func (g *genericExtractor) Extract(ctx context.Context, rawURL string) (any, error) {
	return g.fetch(ctx, rawURL)
}

func matchHost(rawURL string, fn func(host string) bool) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Host)
	host = strings.TrimPrefix(host, "www.")
	host = strings.TrimPrefix(host, "m.")
	return fn(host)
}

// DefaultRegistry creates a pre-populated registry containing all platform extractors.
func DefaultRegistry() *Registry {
	reg := NewRegistry()

	reg.Register(&genericExtractor{
		name: "tiktok",
		matcher: func(h string) bool {
			return h == "tiktok.com" || strings.HasSuffix(h, ".tiktok.com") ||
				h == "douyin.com" || strings.HasSuffix(h, ".douyin.com")
		},
		fetch: func(ctx context.Context, u string) (any, error) {
			return tiktok.FetchData(ctx, u)
		},
	})

	reg.Register(&genericExtractor{
		name: "twitter",
		matcher: func(h string) bool {
			return h == "twitter.com" || h == "x.com" || h == "t.co" ||
				strings.HasSuffix(h, ".twitter.com") || strings.HasSuffix(h, ".x.com")
		},
		fetch: func(ctx context.Context, u string) (any, error) {
			return twitter.FetchData(ctx, u)
		},
	})

	reg.Register(&genericExtractor{
		name: "instagram",
		matcher: func(h string) bool {
			return h == "instagram.com" || h == "instagr.am" || strings.HasSuffix(h, ".instagram.com")
		},
		fetch: func(ctx context.Context, u string) (any, error) {
			return instagram.FetchData(ctx, u)
		},
	})

	reg.Register(&genericExtractor{
		name: "facebook",
		matcher: func(h string) bool {
			return h == "facebook.com" || h == "fb.watch" || h == "fb.gg" || h == "fb.com" ||
				strings.HasSuffix(h, ".facebook.com")
		},
		fetch: func(ctx context.Context, u string) (any, error) {
			return facebook.FetchData(ctx, u)
		},
	})

	reg.Register(&genericExtractor{
		name: "threads",
		matcher: func(h string) bool {
			return h == "threads.net" || h == "threads.com" || strings.HasSuffix(h, ".threads.net")
		},
		fetch: func(ctx context.Context, u string) (any, error) {
			return threads.FetchData(ctx, u)
		},
	})

	reg.Register(&genericExtractor{
		name: "reddit",
		matcher: func(h string) bool {
			return h == "reddit.com" || h == "redd.it" || strings.HasSuffix(h, ".reddit.com")
		},
		fetch: func(ctx context.Context, u string) (any, error) {
			return reddit.FetchData(ctx, u)
		},
	})

	reg.Register(&genericExtractor{
		name: "pinterest",
		matcher: func(h string) bool {
			return h == "pin.it" || strings.Contains(h, "pinterest.")
		},
		fetch: func(ctx context.Context, u string) (any, error) {
			return pinterest.FetchData(ctx, u)
		},
	})

	reg.Register(&genericExtractor{
		name: "terabox",
		matcher: func(h string) bool {
			return strings.Contains(h, "terabox") || strings.Contains(h, "1024tera")
		},
		fetch: func(ctx context.Context, u string) (any, error) {
			return terabox.FetchData(ctx, u)
		},
	})

	reg.Register(&genericExtractor{
		name: "spotify",
		matcher: func(h string) bool {
			return h == "spotify.com" || h == "spoti.fi" || strings.HasSuffix(h, ".spotify.com")
		},
		fetch: func(ctx context.Context, u string) (any, error) {
			return spotify.FetchData(ctx, u)
		},
	})

	reg.Register(&genericExtractor{
		name: "soundcloud",
		matcher: func(h string) bool {
			return h == "soundcloud.com" || h == "on.soundcloud.com" || strings.HasSuffix(h, ".soundcloud.com")
		},
		fetch: func(ctx context.Context, u string) (any, error) {
			return soundcloud.FetchData(ctx, u)
		},
	})

	reg.Register(&genericExtractor{
		name: "capcut",
		matcher: func(h string) bool {
			return strings.Contains(h, "capcut.com")
		},
		fetch: func(ctx context.Context, u string) (any, error) {
			return capcut.FetchData(ctx, u)
		},
	})

	reg.Register(&genericExtractor{
		name: "linkedin",
		matcher: func(h string) bool {
			return h == "linkedin.com" || h == "lnkd.in" || strings.HasSuffix(h, ".linkedin.com")
		},
		fetch: func(ctx context.Context, u string) (any, error) {
			return linkedin.FetchData(ctx, u)
		},
	})

	reg.Register(&genericExtractor{
		name: "pixiv",
		matcher: func(h string) bool {
			return h == "pixiv.net" || h == "pixiv.me" || strings.HasSuffix(h, ".pixiv.net")
		},
		fetch: func(ctx context.Context, u string) (any, error) {
			return pixiv.FetchData(ctx, u)
		},
	})

	return reg
}
