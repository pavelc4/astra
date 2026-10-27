package response_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/response"
)

func TestOK(t *testing.T) {
	rec := httptest.NewRecorder()
	payload := map[string]string{"title": "Video Sample"}
	response.OK(rec, payload, "success download")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var res response.APIResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if !res.Success {
		t.Errorf("expected success true, got false")
	}
	if res.Message != "success download" {
		t.Errorf("expected message %q, got %q", "success download", res.Message)
	}
}

func TestHandleError(t *testing.T) {
	tests := []struct {
		name           string
		inputErr       error
		expectedStatus int
		expectedCode   apperrors.ErrorCode
		expectedMsg    string
	}{
		{
			name: "typed apperror with custom detail",
			inputErr: apperrors.NewCookieExpired(
				"instagram session dead",
				"checkpoint required on account",
			),
			expectedStatus: http.StatusBadGateway,
			expectedCode:   apperrors.CodeCookieExpired,
			expectedMsg:    "instagram session dead",
		},
		{
			name:           "untyped instagram checkpoint/login error (formerly ambiguous 500)",
			inputErr:       errors.New("instagram 403: login/checkpoint required — cookie expired or invalid"),
			expectedStatus: http.StatusBadGateway,
			expectedCode:   apperrors.CodeCookieExpired,
			expectedMsg:    "Upstream authentication required: cookie expired or invalid",
		},
		{
			name:           "untyped rate limit error",
			inputErr:       errors.New("instagram rate-limited (feedback_required) — back off / try later"),
			expectedStatus: http.StatusTooManyRequests,
			expectedCode:   apperrors.CodeRateLimited,
			expectedMsg:    "Upstream rate limit reached, please try again later",
		},
		{
			name:           "untyped not found error",
			inputErr:       errors.New("media not found"),
			expectedStatus: http.StatusNotFound,
			expectedCode:   apperrors.CodeNotFound,
			expectedMsg:    "Requested media or resource not found",
		},
		{
			name:           "untyped invalid url error",
			inputErr:       errors.New("not a valid Pixiv artwork URL"),
			expectedStatus: http.StatusUnprocessableEntity,
			expectedCode:   apperrors.CodeInvalidURL,
			expectedMsg:    "Invalid URL provided",
		},
		{
			name:           "untyped unexpected generic error",
			inputErr:       errors.New("unexpected nil pointer dereference"),
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   apperrors.CodeInternalError,
			expectedMsg:    "Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			response.HandleError(rec, tt.inputErr)

			if rec.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.expectedStatus)
			}

			var res response.APIResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if res.Success {
				t.Errorf("expected success false, got true")
			}
			if res.Message != tt.expectedMsg {
				t.Errorf("message = %q, want %q", res.Message, tt.expectedMsg)
			}
			if res.Error == nil {
				t.Fatalf("expected non-nil Error object")
			}
			if res.Error.Code != tt.expectedCode {
				t.Errorf("error.code = %q, want %q", res.Error.Code, tt.expectedCode)
			}
		})
	}
}
