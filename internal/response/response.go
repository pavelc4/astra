package response

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	apperrors "github.com/pavelc4/astra/internal/errors"
)

type ErrorDetail struct {
	Code   apperrors.ErrorCode `json:"code"`
	Detail string              `json:"detail"`
}

type APIResponse struct {
	Status  int          `json:"status"`
	Success bool         `json:"success"`
	Message string       `json:"message"`
	Data    any          `json:"data,omitempty"`
	Error   *ErrorDetail `json:"error,omitempty"`
}

func OK(w http.ResponseWriter, data any, msg string) {
	writeJSON(w, http.StatusOK, APIResponse{
		Status:  http.StatusOK,
		Success: true,
		Message: msg,
		Data:    data,
	})
}

func Fail(w http.ResponseWriter, status int, msg string, code apperrors.ErrorCode, detail string) {
	writeJSON(w, status, APIResponse{
		Status:  status,
		Success: false,
		Message: msg,
		Error:   &ErrorDetail{Code: code, Detail: detail},
	})
}

func HandleError(w http.ResponseWriter, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		detail := appErr.Detail
		if detail == "" {
			detail = appErr.Message
		}
		Fail(w, appErr.Status, appErr.Message, appErr.Code, detail)
		return
	}

	errMsg := err.Error()
	lowered := strings.ToLower(errMsg)

	switch {
	case strings.Contains(lowered, "login_required") ||
		strings.Contains(lowered, "checkpoint") ||
		strings.Contains(lowered, "cookie expired") ||
		strings.Contains(lowered, "cookie required"):
		Fail(
			w,
			http.StatusBadGateway,
			"Upstream authentication required: cookie expired or invalid",
			apperrors.CodeCookieExpired,
			errMsg,
		)
	case strings.Contains(lowered, "feedback_required") ||
		strings.Contains(lowered, "rate limit") ||
		strings.Contains(lowered, "rate-limit") ||
		strings.Contains(lowered, "too many requests"):
		Fail(
			w,
			http.StatusTooManyRequests,
			"Upstream rate limit reached, please try again later",
			apperrors.CodeRateLimited,
			errMsg,
		)
	case strings.Contains(lowered, "not found") ||
		strings.Contains(lowered, "user not found") ||
		strings.Contains(lowered, "media not found"):
		Fail(
			w,
			http.StatusNotFound,
			"Requested media or resource not found",
			apperrors.CodeNotFound,
			errMsg,
		)
	case strings.Contains(lowered, "invalid url") ||
		strings.Contains(lowered, "not a valid"):
		Fail(
			w,
			http.StatusUnprocessableEntity,
			"Invalid URL provided",
			apperrors.CodeInvalidURL,
			errMsg,
		)
	default:
		Fail(
			w,
			http.StatusInternalServerError,
			"Internal server error",
			apperrors.CodeInternalError,
			errMsg,
		)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to encode json response", "error", err)
	}
}
