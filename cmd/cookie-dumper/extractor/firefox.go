package extractor

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type firefoxPlatformConfig struct {
	hostFilter  string
	cookieNames []string
	label       string
}

var firefoxPlatforms = map[string]firefoxPlatformConfig{
	"facebook": {
		hostFilter:  "%facebook.com",
		cookieNames: []string{"c_user", "xs"},
		label:       "Facebook",
	},
	"pixiv": {
		hostFilter:  "%pixiv.net",
		cookieNames: []string{"PHPSESSID"},
		label:       "Pixiv",
	},
	"instagram": {
		hostFilter:  "%instagram.com",
		cookieNames: []string{"sessionid", "csrftoken"},
		label:       "Instagram",
	},
}

// firefoxExtractor reads cookies directly from the moz_cookies SQLite DB.
// Works with: Firefox, Zen Browser, LibreWolf, Floorp, Waterfox, etc.
type firefoxExtractor struct {
	profileHint string // optional: override profile directory
	platform    string // "instagram", "facebook", or "pixiv"
}

// NewFirefox creates a Firefox/Zen extractor.
// Pass an empty profileHint to use auto-detection.
func NewFirefox(profileHint, platform string) Extractor {
	return &firefoxExtractor{profileHint: profileHint, platform: platform}
}

func (f *firefoxExtractor) Name() string {
	return "Firefox/Zen (SQLite)"
}

func (f *firefoxExtractor) Extract() (*Cookies, error) {
	cfg, ok := firefoxPlatforms[f.platform]
	if !ok {
		cfg = firefoxPlatforms["instagram"]
	}

	dbPath, err := f.resolveCookiesDB()
	if err != nil {
		return nil, fmt.Errorf("resolve cookies.sqlite: %w", err)
	}

	fmt.Printf("[firefox] Using profile: %s\n", filepath.Dir(dbPath))

	// Firefox keeps a WAL lock on the live DB — copy to /tmp first.
	tmpPath, err := copyToTemp(dbPath)
	if err != nil {
		return nil, fmt.Errorf("copy cookies.sqlite to /tmp: %w", err)
	}
	defer os.Remove(tmpPath)

	return queryFirefoxDB(tmpPath, cfg.hostFilter, cfg.cookieNames, cfg.label)
}

// resolveCookiesDB finds the first valid cookies.sqlite on the system.
func (f *firefoxExtractor) resolveCookiesDB() (string, error) {
	if f.profileHint != "" {
		p := filepath.Join(f.profileHint, "cookies.sqlite")

		if _, err := os.Stat(p); err == nil {
			return p, nil
		}

		return "", fmt.Errorf("invalid hint path: %s", f.profileHint)
	}

	u, err := user.Current()
	if err != nil {
		return "", err
	}

	home := u.HomeDir

	globs := []string{
		// Standard Firefox (default)
		filepath.Join(home, ".config", "mozilla", "firefox", "*", "cookies.sqlite"),
		filepath.Join(home, ".mozilla", "firefox", "*", "cookies.sqlite"),

		// ~/.config/zen/ — Zen Browser
		filepath.Join(home, ".config", "zen", "*", "cookies.sqlite"),

		// ~/.zen/ — old location / portable install
		filepath.Join(home, ".zen", "*", "cookies.sqlite"),

		// LibreWolf
		filepath.Join(home, ".librewolf", "*", "cookies.sqlite"),

		// Floorp
		filepath.Join(home, ".floorp", "*", "cookies.sqlite"),

		// snap Firefox
		filepath.Join(
			home,
			"snap",
			"firefox",
			"current",
			".mozilla",
			"firefox",
			"*",
			"cookies.sqlite",
		),

		// macOS
		filepath.Join(
			home,
			"Library",
			"Application Support",
			"Firefox",
			"Profiles",
			"*",
			"cookies.sqlite",
		),

		filepath.Join(
			home,
			"Library",
			"Application Support",
			"Zen",
			"Profiles",
			"*",
			"cookies.sqlite",
		),

		// Windows
		filepath.Join(
			home,
			"AppData",
			"Roaming",
			"Mozilla",
			"Firefox",
			"Profiles",
			"*",
			"cookies.sqlite",
		),
	}

	for _, pattern := range globs {
		matches, err := filepath.Glob(pattern)
		if err != nil || len(matches) == 0 {
			continue
		}

		// Pick the most recently modified profile if multiple exist.
		return newestFile(matches), nil
	}

	return "", fmt.Errorf("cookies.sqlite not found in any known path")
}

func queryFirefoxDB(path, hostFilter string, cookieNames []string, label string) (*Cookies, error) {
	dsn := fmt.Sprintf("file:%s?mode=ro&immutable=1", path)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	defer db.Close()

	query := `SELECT name, value FROM moz_cookies WHERE host LIKE ? ORDER BY lastAccessed DESC`
	rows, err := db.Query(query, hostFilter)
	if err != nil {
		return nil, fmt.Errorf("query moz_cookies: %w", err)
	}
	defer rows.Close()

	vals := make(map[string]string)
	seen := make(map[string]bool)

	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		// Grab every cookie for the host (datr, sb, fr, …); main.go's
		// formatCookieString picks the ones each platform needs. cookieNames
		// below is only the must-be-present check.
		if !seen[name] {
			vals[name] = value
			seen[name] = true
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, cn := range cookieNames {
		if vals[cn] == "" {
			return nil, fmt.Errorf(
				"%s not found — open %s in your browser first, then try again",
				cn, label,
			)
		}
	}

	return &Cookies{Values: vals}, nil
}

func copyToTemp(src string) (string, error) {
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()

	tmp, err := os.CreateTemp("", "ig_cookies_*.sqlite")
	if err != nil {
		return "", err
	}
	defer tmp.Close()

	if _, err := io.Copy(tmp, in); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}

	return tmp.Name(), nil
}

func newestFile(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	best := paths[0]
	var bestTime int64

	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			continue
		}

		if modNano := info.ModTime().UnixNano(); modNano > bestTime {
			bestTime = modNano
			best = p
		}
	}

	return best
}
