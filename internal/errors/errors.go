package errors

import "fmt"

type ErrorCode string

const (
	CodeMissingParam  ErrorCode = "MISSING_PARAMETER"
	CodeInvalidURL    ErrorCode = "INVALID_URL"
	CodeUpstreamError ErrorCode = "UPSTREAM_FAILED"
	CodeNotFound      ErrorCode = "NOT_FOUND"
	CodeInternalError ErrorCode = "INTERNAL_ERROR"
	CodeCookieExpired ErrorCode = "COOKIE_EXPIRED"
	CodeRateLimited   ErrorCode = "RATE_LIMITED"
)

type AppError struct {
	Message string
	Code    ErrorCode
	Status  int
	Detail  string
	Err     error
}

func (e *AppError) Error() string {
	if e.Detail != "" && e.Detail != e.Message {
		return fmt.Sprintf("%s: %s", e.Message, e.Detail)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewDefault(msg string) *AppError {
	return &AppError{Message: msg, Code: CodeInternalError, Status: 500}
}

func NewValidation(msg string) *AppError {
	return &AppError{Message: msg, Code: CodeMissingParam, Status: 400}
}

func NewInvalidURL(msg string) *AppError {
	return &AppError{Message: msg, Code: CodeInvalidURL, Status: 422}
}

func NewUpstream(msg string) *AppError {
	return &AppError{Message: msg, Code: CodeUpstreamError, Status: 502}
}

func NewCookieExpired(msg, detail string) *AppError {
	return &AppError{
		Message: msg,
		Code:    CodeCookieExpired,
		Status:  502,
		Detail:  detail,
	}
}

func NewRateLimited(msg, detail string) *AppError {
	return &AppError{
		Message: msg,
		Code:    CodeRateLimited,
		Status:  429,
		Detail:  detail,
	}
}

func NewNotFound(msg string) *AppError {
	return &AppError{Message: msg, Code: CodeNotFound, Status: 404}
}
