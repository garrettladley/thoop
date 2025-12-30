package xerrors

import (
	"errors"
	"net/http"
	"strings"
	"time"
)

type Error struct {
	StatusCode int
	Message    string
	Cause      error
	RateLimit  *RateLimitInfo
	Validation *ValidationInfo
}

type RateLimitInfo struct {
	RetryAfter time.Duration
	Reason     string
}

type ValidationInfo struct {
	Fields map[string]string
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Cause }

func Unauthorized(opts ...Option) *Error       { return newErr(http.StatusUnauthorized, opts) }
func Forbidden(opts ...Option) *Error          { return newErr(http.StatusForbidden, opts) }
func BadRequest(opts ...Option) *Error         { return newErr(http.StatusBadRequest, opts) }
func NotFound(opts ...Option) *Error           { return newErr(http.StatusNotFound, opts) }
func Internal(opts ...Option) *Error           { return newErr(http.StatusInternalServerError, opts) }
func ServiceUnavailable(opts ...Option) *Error { return newErr(http.StatusServiceUnavailable, opts) }
func TooManyRequests(opts ...Option) *Error { return newErr(http.StatusTooManyRequests, opts) }

func Validation(fields map[string]string, opts ...Option) *Error {
	e := newErr(http.StatusUnprocessableEntity, opts)
	e.Validation = &ValidationInfo{Fields: fields}
	return e
}

func newErr(status int, opts []Option) *Error {
	e := &Error{StatusCode: status, Message: strings.ToLower(http.StatusText(status))}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

type Option func(*Error)

func WithMessage(msg string) Option { return func(e *Error) { e.Message = msg } }
func WithCause(err error) Option    { return func(e *Error) { e.Cause = err } }
func WithRetryAfter(d time.Duration) Option {
	return func(e *Error) {
		if e.RateLimit == nil {
			e.RateLimit = &RateLimitInfo{}
		}
		e.RateLimit.RetryAfter = d
	}
}

func WithReason(reason string) Option {
	return func(e *Error) {
		if e.RateLimit == nil {
			e.RateLimit = &RateLimitInfo{}
		}
		e.RateLimit.Reason = reason
	}
}

func As(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return nil
}
