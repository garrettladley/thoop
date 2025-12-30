package xhttp

import (
	"net/http"

	"github.com/garrettladley/thoop/internal/version"
)

type thoopTransport struct {
	base http.RoundTripper
}

func (t *thoopTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", "thoop/"+version.Get())
	req.Header.Set(version.Header, version.Get())
	return t.base.RoundTrip(req)
}

// NewTransport returns an http.RoundTripper with standard thoop headers.
func NewTransport() http.RoundTripper {
	return &thoopTransport{base: http.DefaultTransport}
}
