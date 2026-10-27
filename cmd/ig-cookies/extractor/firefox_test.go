package extractor

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestFirefoxPlatformsConfig(t *testing.T) {
	tests := []struct {
		platform    string
		wantHost    string
		wantLabel   string
		mustCookies []string
	}{
		{"instagram", "%instagram.com", "Instagram", []string{"sessionid", "csrftoken"}},
		{"facebook", "%facebook.com", "Facebook", []string{"c_user", "xs"}},
		{"pixiv", "%pixiv.net", "Pixiv", []string{"PHPSESSID"}},
	}

	for _, tt := range tests {
		cfg, ok := firefoxPlatforms[tt.platform]
		if !ok {
			t.Fatalf("expected platform %s to be registered", tt.platform)
		}
		if cfg.hostFilter != tt.wantHost {
			t.Errorf("[%s] hostFilter = %s, want %s", tt.platform, cfg.hostFilter, tt.wantHost)
		}
		if cfg.label != tt.wantLabel {
			t.Errorf("[%s] label = %s, want %s", tt.platform, cfg.label, tt.wantLabel)
		}
		for _, wantCookie := range tt.mustCookies {
			found := false
			for _, cn := range cfg.cookieNames {
				if cn == wantCookie {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("[%s] missing required cookie %s", tt.platform, wantCookie)
			}
		}
	}
}

func TestQueryFirefoxDB(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "cookies.sqlite")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE moz_cookies (
			id INTEGER PRIMARY KEY,
			name TEXT,
			value TEXT,
			host TEXT,
			lastAccessed INTEGER
		);
		INSERT INTO moz_cookies (name, value, host, lastAccessed) VALUES
			('sessionid', 'test_session_val', '.instagram.com', 100),
			('csrftoken', 'test_csrf_val', '.instagram.com', 100),
			('ds_user_id', '12345678', '.instagram.com', 100),
			('other', 'foo', '.other.com', 50);
	`)
	if err != nil {
		db.Close()
		t.Fatalf("failed to setup test db: %v", err)
	}
	db.Close()

	// Test successful extraction
	cookies, err := queryFirefoxDB(dbPath, "%instagram.com", []string{"sessionid", "csrftoken"}, "Instagram")
	if err != nil {
		t.Fatalf("queryFirefoxDB failed: %v", err)
	}

	if cookies.Values["sessionid"] != "test_session_val" {
		t.Errorf("expected sessionid=test_session_val, got %s", cookies.Values["sessionid"])
	}
	if cookies.Values["csrftoken"] != "test_csrf_val" {
		t.Errorf("expected csrftoken=test_csrf_val, got %s", cookies.Values["csrftoken"])
	}

	// Test missing required cookie
	_, err = queryFirefoxDB(dbPath, "%instagram.com", []string{"missing_cookie"}, "Instagram")
	if err == nil {
		t.Fatal("expected error for missing required cookie, got nil")
	}
}

func TestNewestFile(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	_ = os.WriteFile(file1, []byte("old"), 0644)
	time.Sleep(20 * time.Millisecond)
	_ = os.WriteFile(file2, []byte("new"), 0644)

	newest := newestFile([]string{file1, file2})
	if newest != file2 {
		t.Errorf("expected newestFile to be %s, got %s", file2, newest)
	}
}
