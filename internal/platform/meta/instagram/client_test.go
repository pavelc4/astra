package instagram

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "github.com/pavelc4/astra/internal/errors"
)

func TestShortcodeToMediaID(t *testing.T) {
	tests := []struct {
		name      string
		shortcode string
		wantID    string
		wantErr   bool
	}{
		{
			name:      "valid shortcode 1",
			shortcode: "B_uY9Znp-zE",
			wantErr:   false,
		},
		{
			name:      "invalid character",
			shortcode: "invalid@char",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := shortcodeToMediaID(tt.shortcode)
			if (err != nil) != tt.wantErr {
				t.Fatalf("shortcodeToMediaID() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got == "" {
				t.Errorf("expected non-empty media ID")
			}
		})
	}
}

func TestIGClient_GetJSON_ErrorClassification(t *testing.T) {
	tests := []struct {
		name         string
		serverStatus int
		serverBody   string
		expectedCode apperrors.ErrorCode
	}{
		{
			name:         "checkpoint / login_required",
			serverStatus: http.StatusBadRequest,
			serverBody:   `{"message": "checkpoint_required", "status": "fail"}`,
			expectedCode: apperrors.CodeCookieExpired,
		},
		{
			name:         "rate-limited / feedback_required",
			serverStatus: http.StatusBadRequest,
			serverBody:   `{"message": "feedback_required", "status": "fail"}`,
			expectedCode: apperrors.CodeRateLimited,
		},
		{
			name:         "generic upstream error",
			serverStatus: http.StatusInternalServerError,
			serverBody:   `{"message": "server error", "status": "fail"}`,
			expectedCode: apperrors.CodeUpstreamError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.serverStatus)
				_, _ = w.Write([]byte(tt.serverBody))
			}))
			defer server.Close()

			client := NewIGClient("sessionid=mock123")
			client.http = server.Client()

			_, err := client.getJSON(context.Background(), server.URL)
			if err == nil {
				t.Fatalf("expected error from getJSON, got nil")
			}

			var appErr *apperrors.AppError
			if !errors.As(err, &appErr) {
				t.Fatalf("expected *apperrors.AppError in error chain, got %T: %v", err, err)
			}

			if appErr.Code != tt.expectedCode {
				t.Errorf("expected code %s, got %s", tt.expectedCode, appErr.Code)
			}
		})
	}
}
