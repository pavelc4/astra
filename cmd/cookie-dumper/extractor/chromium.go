package extractor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"time"

	"github.com/chromedp/cdproto/storage"
	"github.com/chromedp/chromedp"
)

// ChromiumExtractor launches a Chromium-based browser via CDP and extracts
// cookies interactively.
type ChromiumExtractor struct {
	ProfilePath string
	ExecPath    string
	Platform    string // "instagram" or "facebook"
}

// NewChromiumExtractor returns a ChromiumExtractor.
// Pass empty strings to auto-detect the profile and browser binary
// (Chrome, Chromium, Brave, Edge, Vivaldi).
func NewChromiumExtractor(profilePath, execPath, platform string) (*ChromiumExtractor, error) {
	if profilePath == "" {
		p, err := detectChromiumProfile()
		if err != nil {
			return nil, err
		}
		profilePath = p
	}

	if execPath == "" {
		e, err := detectChromiumExec()
		if err != nil {
			return nil, err
		}
		execPath = e
	}

	return &ChromiumExtractor{
		ProfilePath: profilePath,
		ExecPath:    execPath,
		Platform:    platform,
	}, nil
}

func (c *ChromiumExtractor) targetDomain() (domain, label string) {
	switch c.Platform {
	case "facebook":
		return "https://www.facebook.com/", "Facebook"
	default:
		return "https://www.instagram.com/", "Instagram"
	}
}

func (c *ChromiumExtractor) requiredCookies() []string {
	switch c.Platform {
	case "facebook":
		return []string{"c_user", "xs"}
	default:
		return []string{"sessionid", "csrftoken"}
	}
}

// Extract launches the browser, navigates to the target platform, waits for
// the user to confirm login, then retrieves session cookies via CDP.
func (c *ChromiumExtractor) Extract() (*Cookies, error) {
	profile := c.ProfilePath

	if isProfileLocked(c.ProfilePath) {
		tmpProfile, err := os.MkdirTemp("", "ig-chrome-profile-*")
		if err != nil {
			return nil, fmt.Errorf("create temp profile: %w", err)
		}
		defer os.RemoveAll(tmpProfile)

		profile = tmpProfile

		fmt.Printf("The original profile is locked by a running Chrome instance.\n")
		fmt.Printf("Using a temporary profile — you’ll need to log in first.\n")
		fmt.Println()
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.UserDataDir(profile),
		chromedp.ExecPath(c.ExecPath),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	domain, label := c.targetDomain()

	fmt.Printf("Using browser binary : %s\n", c.ExecPath)
	fmt.Printf("Using profile        : %s\n", profile)
	fmt.Println()
	fmt.Printf("The browser will open. Log in to %s, then press ENTER.\n", label)
	fmt.Println()

	if err := chromedp.Run(ctx,
		chromedp.Navigate(domain),
	); err != nil {
		return nil, fmt.Errorf("failed to navigate to %s: %w", label, err)
	}

	fmt.Print("Press ENTER once you are logged in... ")
	fmt.Scanln()

	var result *Cookies

	err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		cdpCookies, err := storage.GetCookies().Do(ctx)
		if err != nil {
			return fmt.Errorf("get cookies: %w", err)
		}

		vals := make(map[string]string)
		for _, co := range cdpCookies {
			vals[co.Name] = co.Value
		}

		for _, name := range c.requiredCookies() {
			if vals[name] == "" {
				return fmt.Errorf(
					"%s not found — make sure you are logged into %s",
					name, label,
				)
			}
		}

		result = &Cookies{Values: vals}
		return nil
	}))

	if err != nil {
		return nil, err
	}

	return result, nil
}

// isProfileLocked checks whether a Chrome profile directory contains a
// SingletonLock file — a sign that another Chrome instance is using it.
func isProfileLocked(path string) bool {
	lock := filepath.Join(path, "SingletonLock")

	_, err := os.Stat(lock)
	return err == nil
}

// --------------------------------------------------------------------------
// Profile & binary detection
// --------------------------------------------------------------------------

type chromiumBrowser struct {
	name    string
	exec    []string
	profile string
}

func chromiumBrowserList(home string) []chromiumBrowser {
	return []chromiumBrowser{
		{
			name:    "Google Chrome",
			exec:    []string{"google-chrome", "google-chrome-stable"},
			profile: filepath.Join(home, ".config/google-chrome"),
		},
		{
			name:    "Chromium",
			exec:    []string{"chromium", "chromium-browser"},
			profile: filepath.Join(home, ".config/chromium"),
		},
		{
			name:    "Brave",
			exec:    []string{"brave-browser", "brave"},
			profile: filepath.Join(home, ".config/BraveSoftware/Brave-Browser"),
		},
		{
			name:    "Microsoft Edge",
			exec:    []string{"microsoft-edge", "microsoft-edge-stable"},
			profile: filepath.Join(home, ".config/microsoft-edge"),
		},
		{
			name:    "Vivaldi",
			exec:    []string{"vivaldi", "vivaldi-stable"},
			profile: filepath.Join(home, ".config/vivaldi"),
		},

		// macOS paths
		{
			name: "Google Chrome (macOS)",
			exec: []string{
				"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			},
			profile: filepath.Join(
				home,
				"Library/Application Support/Google/Chrome",
			),
		},
		{
			name: "Brave (macOS)",
			exec: []string{
				"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
			},
			profile: filepath.Join(
				home,
				"Library/Application Support/BraveSoftware/Brave-Browser",
			),
		},
		{
			name: "Microsoft Edge (macOS)",
			exec: []string{
				"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			},
			profile: filepath.Join(
				home,
				"Library/Application Support/Microsoft Edge",
			),
		},
	}
}

func detectChromiumProfile() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	for _, b := range chromiumBrowserList(u.HomeDir) {
		if _, err := os.Stat(b.profile); err == nil {
			fmt.Printf("Detected %s profile: %s\n", b.name, b.profile)
			return b.profile, nil
		}
	}

	return "", fmt.Errorf(
		"no Chromium-based browser profile found\n" +
			"Manual override: -profile /path/to/profile -exec /path/to/browser",
	)
}

func detectChromiumExec() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	for _, b := range chromiumBrowserList(u.HomeDir) {
		for _, e := range b.exec {
			// Check PATH first
			if p, err := exec.LookPath(e); err == nil {
				return p, nil
			}

			// Then check absolute path
			if _, err := os.Stat(e); err == nil {
				return e, nil
			}
		}
	}

	return "", fmt.Errorf(
		"no Chromium-based browser binary found\n" +
			"Manual override: -exec /path/to/browser",
	)
}
