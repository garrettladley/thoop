package xhttp

import (
	"fmt"
	"net/http"

	"github.com/garrettladley/thoop/internal/version"
)

type thoopTransport struct {
	base http.RoundTripper
}

var _ http.RoundTripper = (*thoopTransport)(nil)

func (t *thoopTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", "thoop/"+version.Get())
	req.Header.Set(version.Header, version.Get())
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform round trip: %w", err)
	}
	return resp, nil
}

// NewTransport returns an http.RoundTripper with standard thoop headers.
func NewTransport() http.RoundTripper {
	return &thoopTransport{base: http.DefaultTransport}
}
