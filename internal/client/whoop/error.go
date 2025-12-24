package whoop

import (
	"fmt"
	"io"
	"net/http"

	go_json "github.com/goccy/go-json"
)

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("whoop api: %d %s", e.StatusCode, e.Message)
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

	return &APIError{
		StatusCode: resp.StatusCode,
		Message:    msg,
	}
}
