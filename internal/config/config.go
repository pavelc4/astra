package config

import (
	"bufio"
	"os"
	"strings"

	"github.com/pavelc4/astra/internal/platform/meta/facebook"
	"github.com/pavelc4/astra/internal/platform/meta/instagram"
	"github.com/pavelc4/astra/internal/platform/pixiv"
)

type Config struct {
	Port string
	Host string
}

func Load() Config {
	loadDotEnv(".env")

	cfg := Config{
		Port: getEnv("PORT", "3000"),
		Host: getEnv("HOST", "0.0.0.0"),
	}

	// Always load the latest cookies from .env into memory
	_, _ = ReloadCookies()

	return cfg
}

// ReloadCookies re-reads the .env file and updates platform cookies in-memory without restart.
func ReloadCookies() ([]string, error) {
	f, err := os.Open(".env")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var reloaded []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)

		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') ||
				(val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}

		switch key {
		case "INSTAGRAM_COOKIES":
			os.Setenv(key, val)
			instagram.SetCookies(val)
			reloaded = append(reloaded, "instagram")
		case "FACEBOOK_COOKIES":
			os.Setenv(key, val)
			facebook.SetCookies(val)
			reloaded = append(reloaded, "facebook")
		case "PIXIV_COOKIES":
			os.Setenv(key, val)
			pixiv.SetCookies(val)
			reloaded = append(reloaded, "pixiv")
		}
	}

	return reloaded, scanner.Err()
}

// loadDotEnv reads key=value pairs from a .env file and sets them as
// environment variables — skips keys that are already set in the environment
// so that real env vars always take precedence.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return // nggak ada .env? fine, skip aja
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines dan komentar
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)

		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') ||
				(val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}

		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
