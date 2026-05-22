package xhttp

import (
	"net/http"
	"testing"
)

const (
	testForwardedIPv4      = "203.0.113.195"
	testRemoteIPv4         = "192.0.2.1"
	testRemoteIPv4WithPort = testRemoteIPv4 + ":1234"
	testForwardedIPv6      = "2001:db8::1"
)

func TestGetIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		xForwardedFor string
		remoteAddr    string
		expectedIP    string
	}{
		{
			name:          "x-forwarded-for with IP only",
			xForwardedFor: testForwardedIPv4,
			remoteAddr:    testRemoteIPv4WithPort,
			expectedIP:    testForwardedIPv4,
		},
		{
			name:          "x-forwarded-for with IP and port",
			xForwardedFor: "203.0.113.195:8080",
			remoteAddr:    testRemoteIPv4WithPort,
			expectedIP:    testForwardedIPv4,
		},
		{
			name:          "remote addr with IP and port",
			xForwardedFor: "",
			remoteAddr:    testRemoteIPv4WithPort,
			expectedIP:    testRemoteIPv4,
		},
		{
			name:          "remote addr with IP only",
			xForwardedFor: "",
			remoteAddr:    testRemoteIPv4,
			expectedIP:    testRemoteIPv4,
		},
		{
			name:          "IPv6 in x-forwarded-for",
			xForwardedFor: testForwardedIPv6,
			remoteAddr:    testRemoteIPv4WithPort,
			expectedIP:    testForwardedIPv6,
		},
		{
			name:          "IPv6 with port in x-forwarded-for",
			xForwardedFor: "[2001:db8::1]:8080",
			remoteAddr:    testRemoteIPv4WithPort,
			expectedIP:    testForwardedIPv6,
		},
		{
			name:          "IPv6 in remote addr",
			xForwardedFor: "",
			remoteAddr:    "[2001:db8::1]:1234",
			expectedIP:    testForwardedIPv6,
		},
		{
			name:          "IPv6 without port in remote addr",
			xForwardedFor: "",
			remoteAddr:    testForwardedIPv6,
			expectedIP:    testForwardedIPv6,
		},
		{
			name:          "localhost IPv4",
			xForwardedFor: "",
			remoteAddr:    "127.0.0.1:8080",
			expectedIP:    "127.0.0.1",
		},
		{
			name:          "localhost IPv6",
			xForwardedFor: "",
			remoteAddr:    "[::1]:8080",
			expectedIP:    "::1",
		},
		{
			name:          "x-forwarded-for takes precedence",
			xForwardedFor: testForwardedIPv4,
			remoteAddr:    testRemoteIPv4WithPort,
			expectedIP:    testForwardedIPv4,
		},
		{
			name:          "empty remote addr",
			xForwardedFor: "",
			remoteAddr:    "",
			expectedIP:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := buildRequest(t, tt.xForwardedFor, tt.remoteAddr)
			got := GetRequestIP(req)

			if got != tt.expectedIP {
				t.Errorf("GetIP() = %q, want %q", got, tt.expectedIP)
			}
		})
	}
}

func buildRequest(t *testing.T, xForwardedFor, remoteAddr string) *http.Request {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	if xForwardedFor != "" {
		req.Header.Set(XForwardedFor, xForwardedFor)
	}

	req.RemoteAddr = remoteAddr

	return req
}
