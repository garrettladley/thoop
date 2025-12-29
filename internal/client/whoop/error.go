package whoop

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	go_json "github.com/goccy/go-json"
)

type APIError struct {
	StatusCode int
	Message    string
	RetryAfter time.Duration
}

func (e *APIError) Error() string {
	return fmt.Sprintf("whoop api: %d %s", e.StatusCode, e.Message)
}

func (e *APIError) IsRateLimited() bool {
	return e.StatusCode == http.StatusTooManyRequests
}

func parseAPIError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    resp.Status,
		}
	}

	var errResp struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}

	if err := go_json.Unmarshal(body, &errResp); err != nil {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	msg := errResp.Message
	if msg == "" {
		msg = errResp.Error
	}
	if msg == "" {
		msg = resp.Status
	}

	apiErr := &APIError{
		StatusCode: resp.StatusCode,
		Message:    msg,
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		apiErr.RetryAfter = parseRetryAfter(resp.Header.Get("Retry-After"))
	}

	return apiErr
}

func parseRetryAfter(value string) time.Duration {
	const minRetryAfter = 5 * time.Second

	if value == "" {
		return minRetryAfter
	}
	seconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return minRetryAfter
	}
	d := time.Duration(seconds) * time.Second
	if d < minRetryAfter {
		return minRetryAfter
	}
	return d
}
