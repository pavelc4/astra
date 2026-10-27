package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pavelc4/astra/internal/config"
)

func TestLoad(t *testing.T) {
	cfg := config.Load()
	if cfg.Port == "" {
		t.Errorf("expected default port")
	}
	if cfg.Host == "" {
		t.Errorf("expected default host")
	}
}

func TestReloadCookies(t *testing.T) {
	// Create temporary directory and change working directory to it
	tempDir := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get wd: %v", err)
	}
	defer func() {
		_ = os.Chdir(origWd)
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to chdir to tempDir: %v", err)
	}

	envContent := `
# Temporary test env
INSTAGRAM_COOKIES="sessionid=fresh_test_session_123"
FACEBOOK_COOKIES="c_user=test_user; xs=test_xs"
PIXIV_COOKIES="PHPSESSID=test_pixiv_session"
`
	if err := os.WriteFile(filepath.Join(tempDir, ".env"), []byte(envContent), 0o600); err != nil {
		t.Fatalf("failed to write .env: %v", err)
	}

	reloaded, err := config.ReloadCookies()
	if err != nil {
		t.Fatalf("ReloadCookies failed: %v", err)
	}

	if len(reloaded) < 3 {
		t.Errorf("expected at least 3 platforms reloaded, got %v", reloaded)
	}

	if os.Getenv("INSTAGRAM_COOKIES") != "sessionid=fresh_test_session_123" {
		t.Errorf("INSTAGRAM_COOKIES was not updated")
	}
	if os.Getenv("PIXIV_COOKIES") != "PHPSESSID=test_pixiv_session" {
		t.Errorf("PIXIV_COOKIES was not updated")
	}
}
