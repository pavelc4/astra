package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pavelc4/astra/cmd/cookie-dumper/extractor"
)

func TestFormatCookieString(t *testing.T) {
	cookies := &extractor.Cookies{
		Values: map[string]string{
			"sessionid":    "sess_123",
			"csrftoken":    "csrf_456",
			"c_user":       "user_789",
			"xs":           "xs_abc",
			"datr":         "datr_def",
			"sb":           "sb_ghi",
			"fr":           "fr_jkl",
			"dpr":          "1.0",
			"PHPSESSID":    "php_mno",
			"login_ever":   "yes",
			"device_token": "token_pqr",
			"p_ab_id":      "0",
			"cf_clearance": "cf_stu",
		},
	}

	igFormatted := formatCookieString(cookies, platformConfigs["instagram"])
	if igFormatted != "sessionid=sess_123; csrftoken=csrf_456" {
		t.Errorf("unexpected instagram cookies: %s", igFormatted)
	}

	fbFormatted := formatCookieString(cookies, platformConfigs["facebook"])
	expectedFB := "c_user=user_789; xs=xs_abc; datr=datr_def; sb=sb_ghi; fr=fr_jkl; dpr=1.0"
	if fbFormatted != expectedFB {
		t.Errorf("unexpected facebook cookies: %s", fbFormatted)
	}

	pixivFormatted := formatCookieString(cookies, platformConfigs["pixiv"])
	expectedPixiv := "PHPSESSID=php_mno; login_ever=yes; device_token=token_pqr; p_ab_id=0; cf_clearance=cf_stu"
	if pixivFormatted != expectedPixiv {
		t.Errorf("unexpected pixiv cookies: %s", pixivFormatted)
	}
}

func TestCreateExtractor(t *testing.T) {
	ex, err := createExtractor("firefox", "", "", "instagram")
	if err != nil {
		t.Fatalf("createExtractor(firefox) failed: %v", err)
	}
	if ex == nil {
		t.Fatal("expected non-nil extractor")
	}

	_, err = createExtractor("invalid_browser", "", "", "instagram")
	if err == nil {
		t.Fatal("expected error for invalid browser, got nil")
	}
}

func TestUpdateEnvFile(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")

	initialContent := "FOO=\"bar\"\nINSTAGRAM_COOKIES=\"old_val\"\nBAZ=\"qux\"\n"
	if err := os.WriteFile(envPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("failed to write initial env: %v", err)
	}

	// Update existing key
	if err := updateEnvFile(envPath, "INSTAGRAM_COOKIES", "new_val"); err != nil {
		t.Fatalf("updateEnvFile failed: %v", err)
	}

	updated, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("failed to read updated env: %v", err)
	}
	content := string(updated)

	if !strings.Contains(content, "INSTAGRAM_COOKIES=\"new_val\"") {
		t.Errorf("expected updated INSTAGRAM_COOKIES, got:\n%s", content)
	}
	if !strings.Contains(content, "FOO=\"bar\"") || !strings.Contains(content, "BAZ=\"qux\"") {
		t.Errorf("other keys were unexpectedly modified:\n%s", content)
	}

	// Append new key
	if err := updateEnvFile(envPath, "NEW_KEY", "hello"); err != nil {
		t.Fatalf("append key failed: %v", err)
	}
	appended, _ := os.ReadFile(envPath)
	if !strings.Contains(string(appended), "NEW_KEY=\"hello\"") {
		t.Errorf("expected appended key, got:\n%s", string(appended))
	}
}
