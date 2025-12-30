package apperr

import (
	"errors"
	"net/http"
	"time"
)

type Error struct {
	Code       string
	Message    string
	StatusCode int
	Cause      error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Cause
}

type RateLimitError struct {
	Code       string
	Message    string
	StatusCode int
	Cause      error
	RetryAfter time.Duration
	Reason     string
}

func (e *RateLimitError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *RateLimitError) Unwrap() error {
	return e.Cause
}

func Unauthorized(code, message string) *Error {
	return &Error{Code: code, Message: message, StatusCode: http.StatusUnauthorized}
}

func Forbidden(code, message string) *Error {
	return &Error{Code: code, Message: message, StatusCode: http.StatusForbidden}
}

func BadRequest(code, message string) *Error {
	return &Error{Code: code, Message: message, StatusCode: http.StatusBadRequest}
}

func NotFound(code, message string) *Error {
	return &Error{Code: code, Message: message, StatusCode: http.StatusNotFound}
}

func Internal(code, message string, cause error) *Error {
	return &Error{Code: code, Message: message, StatusCode: http.StatusInternalServerError, Cause: cause}
}

func TooManyRequests(code, message string, retryAfter time.Duration, reason string) *RateLimitError {
	return &RateLimitError{
		Code:       code,
		Message:    message,
		StatusCode: http.StatusTooManyRequests,
		RetryAfter: retryAfter,
		Reason:     reason,
	}
}

func ServiceUnavailable(code, message string) *Error {
	return &Error{Code: code, Message: message, StatusCode: http.StatusServiceUnavailable}
}

func AsError(err error) *Error {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr
	}
	var rlErr *RateLimitError
	if errors.As(err, &rlErr) {
		return &Error{
			Code:       rlErr.Code,
			Message:    rlErr.Message,
			StatusCode: rlErr.StatusCode,
			Cause:      rlErr.Cause,
		}
	}
	return nil
}

func AsRateLimitError(err error) *RateLimitError {
	var rlErr *RateLimitError
	if errors.As(err, &rlErr) {
		return rlErr
	}
	return nil
}
