package xhttp

import (
	"net/http"
	"time"
)

type ClientOption func(*http.Client)

func WithTimeout(d time.Duration) ClientOption {
	return func(c *http.Client) { c.Timeout = d }
}

func NewHTTPClient(opts ...ClientOption) *http.Client {
	c := &http.Client{Transport: NewTransport()}
	for _, opt := range opts {
		opt(c)
	}
	return c
}
