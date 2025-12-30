package xhttp

import (
	"fmt"
	"net/http"
	"time"
)

const (
	XForwardedFor    = "X-Forwarded-For"
	XContentTypeOpts = "X-Content-Type-Options"
	XFrameOpts       = "X-Frame-Options"
	XXSSProtection   = "X-Xss-Protection"
	ReferrerPolicy   = "Referrer-Policy"
)

const (
	ContentType      = "Content-Type"
	ContentLength    = "Content-Length"
	ContentEncoding  = "Content-Encoding"
	AcceptEncoding   = "Accept-Encoding"
	Vary             = "Vary"
	XRateLimitReason = "X-Ratelimit-Reason"
	XSessionID       = "X-Session-Id"
	XAPIKey          = "X-Api-Key" //nolint:gosec // this is a header name, not a credential
)

func SetHeaderRequestID(w http.ResponseWriter, requestID string) {
	const headerName = "X-Request-ID"
	w.Header().Set(headerName, requestID)
}

func SetHeaderContentTypeApplicationJSON(w http.ResponseWriter) {
	const applicationJSON = "application/json"
	w.Header().Set(ContentType, applicationJSON)
}

func SetHeaderContentTypeTextHTML(w http.ResponseWriter) {
	const textHTML = "text/html"
	w.Header().Set(ContentType, textHTML)
}

func SetHeaderRetryAfter(w http.ResponseWriter, retryAfter time.Duration) {
	const retryAfterHeader = "Retry-After"
	retryAfterSeconds := int(retryAfter.Seconds())
	w.Header().Set(retryAfterHeader, fmt.Sprintf("%d", retryAfterSeconds))
}

func SetRequestHeaderSessionID(req *http.Request, sessionID string) {
	req.Header.Set(XSessionID, sessionID)
}

func GetRequestHeaderSessionID(req *http.Request) string {
	return req.Header.Get(XSessionID)
}

func GetRequestHeaderAPIKey(req *http.Request) string {
	return req.Header.Get(XAPIKey)
}
