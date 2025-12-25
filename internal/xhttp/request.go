package xhttp

import (
	"net"
	"net/http"
)

func GetRequestIP(r *http.Request) string {
	if xff := r.Header.Get(XForwardedFor); xff != "" {
		if ip, _, err := net.SplitHostPort(xff); err == nil {
			return ip
		}
		return xff
	}
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return ip
	}
	return r.RemoteAddr
}
