package xhttp

import (
	"net/http"
	"time"

	go_json "github.com/goccy/go-json"
)

func Error(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

type RateLimitError struct {
	RetryAfter time.Duration
	Reason     string
	Message    string
}

func (e *RateLimitError) Error() string {
	return "rate limited"
}

func WriteRateLimitError(w http.ResponseWriter, err *RateLimitError) {
	SetHeaderContentTypeApplicationJSON(w)
	SetHeaderRetryAfter(w, err.RetryAfter)
	if err.Reason != "" {
		w.Header().Set(XRateLimitReason, err.Reason)
	}
	status := http.StatusTooManyRequests
	w.WriteHeader(status)

	msg := err.Message
	if msg == "" {
		msg = http.StatusText(status)
	}

	_ = go_json.NewEncoder(w).Encode(map[string]string{
		"error":   "rate_limited",
		"message": msg,
	})
}
