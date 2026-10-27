package errors_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	apperrors "github.com/pavelc4/astra/internal/errors"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		appErr   *apperrors.AppError
		expected string
	}{
		{
			name: "message only",
			appErr: &apperrors.AppError{
				Message: "something went wrong",
			},
			expected: "something went wrong",
		},
		{
			name: "message and detail identical",
			appErr: &apperrors.AppError{
				Message: "validation error",
				Detail:  "validation error",
			},
			expected: "validation error",
		},
		{
			name: "message and distinct detail",
			appErr: &apperrors.AppError{
				Message: "cookie expired",
				Detail:  "instagram checkpoint required",
			},
			expected: "cookie expired: instagram checkpoint required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.appErr.Error()
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	rootErr := fmt.Errorf("root cause")
	wrapped := &apperrors.AppError{
		Message: "failed operation",
		Err:     rootErr,
	}

	if !errors.Is(wrapped, rootErr) {
		t.Fatalf("expected errors.Is to find root cause in chain")
	}
}

func TestConstructors(t *testing.T) {
	tests := []struct {
		name           string
		err            *apperrors.AppError
		expectedStatus int
		expectedCode   apperrors.ErrorCode
		expectedMsg    string
	}{
		{
			name:           "NewDefault",
			err:            apperrors.NewDefault("server crash"),
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   apperrors.CodeInternalError,
			expectedMsg:    "server crash",
		},
		{
			name:           "NewValidation",
			err:            apperrors.NewValidation("missing param"),
			expectedStatus: http.StatusBadRequest,
			expectedCode:   apperrors.CodeMissingParam,
			expectedMsg:    "missing param",
		},
		{
			name:           "NewInvalidURL",
			err:            apperrors.NewInvalidURL("bad url"),
			expectedStatus: http.StatusUnprocessableEntity,
			expectedCode:   apperrors.CodeInvalidURL,
			expectedMsg:    "bad url",
		},
		{
			name:           "NewUpstream",
			err:            apperrors.NewUpstream("upstream timeout"),
			expectedStatus: http.StatusBadGateway,
			expectedCode:   apperrors.CodeUpstreamError,
			expectedMsg:    "upstream timeout",
		},
		{
			name:           "NewCookieExpired",
			err:            apperrors.NewCookieExpired("session dead", "checkpoint"),
			expectedStatus: http.StatusBadGateway,
			expectedCode:   apperrors.CodeCookieExpired,
			expectedMsg:    "session dead",
		},
		{
			name:           "NewRateLimited",
			err:            apperrors.NewRateLimited("too many requests", "feedback_required"),
			expectedStatus: http.StatusTooManyRequests,
			expectedCode:   apperrors.CodeRateLimited,
			expectedMsg:    "too many requests",
		},
		{
			name:           "NewNotFound",
			err:            apperrors.NewNotFound("media missing"),
			expectedStatus: http.StatusNotFound,
			expectedCode:   apperrors.CodeNotFound,
			expectedMsg:    "media missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Status != tt.expectedStatus {
				t.Errorf("status = %d, want %d", tt.err.Status, tt.expectedStatus)
			}
			if tt.err.Code != tt.expectedCode {
				t.Errorf("code = %s, want %s", tt.err.Code, tt.expectedCode)
			}
			if tt.err.Message != tt.expectedMsg {
				t.Errorf("message = %q, want %q", tt.err.Message, tt.expectedMsg)
			}
		})
	}
}
