package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/garrettladley/thoop/internal/xhttp"
)

func TestGzip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		acceptEncoding     string
		path               string
		responseBody       string
		existingEncoding   string
		wantCompressed     bool
		wantVaryHeader     bool
		wantContentEncoded bool
	}{
		{
			name:               "small response not compressed",
			acceptEncoding:     "gzip",
			path:               "/api/test",
			responseBody:       "small",
			wantCompressed:     false,
			wantVaryHeader:     true,
			wantContentEncoded: false,
		},
		{
			name:               "large response compressed",
			acceptEncoding:     "gzip",
			path:               "/api/test",
			responseBody:       strings.Repeat("x", 2000),
			wantCompressed:     true,
			wantVaryHeader:     true,
			wantContentEncoded: true,
		},
		{
			name:               "exactly 1KB compressed",
			acceptEncoding:     "gzip",
			path:               "/api/test",
			responseBody:       strings.Repeat("y", 1024),
			wantCompressed:     true,
			wantVaryHeader:     true,
			wantContentEncoded: true,
		},
		{
			name:               "no accept-encoding header",
			acceptEncoding:     "",
			path:               "/api/test",
			responseBody:       strings.Repeat("x", 2000),
			wantCompressed:     false,
			wantVaryHeader:     false,
			wantContentEncoded: false,
		},
		{
			name:               "accept-encoding without gzip",
			acceptEncoding:     "deflate, br",
			path:               "/api/test",
			responseBody:       strings.Repeat("x", 2000),
			wantCompressed:     false,
			wantVaryHeader:     false,
			wantContentEncoded: false,
		},
		{
			name:               "SSE path excluded",
			acceptEncoding:     "gzip",
			path:               "/api/notifications/stream",
			responseBody:       strings.Repeat("x", 2000),
			wantCompressed:     false,
			wantVaryHeader:     false,
			wantContentEncoded: false,
		},
		{
			name:               "already encoded response",
			acceptEncoding:     "gzip",
			path:               "/api/test",
			responseBody:       strings.Repeat("x", 2000),
			existingEncoding:   "br",
			wantCompressed:     false,
			wantVaryHeader:     true,
			wantContentEncoded: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if tt.existingEncoding != "" {
					w.Header().Set(xhttp.ContentEncoding, tt.existingEncoding)
				}
				_, _ = w.Write([]byte(tt.responseBody))
			})

			wrapped := Gzip(handler)

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tt.path, nil)
			if tt.acceptEncoding != "" {
				req.Header.Set(xhttp.AcceptEncoding, tt.acceptEncoding)
			}

			rec := httptest.NewRecorder()
			wrapped.ServeHTTP(rec, req)

			resp := rec.Result()
			defer resp.Body.Close() //nolint:errcheck

			gotVary := resp.Header.Get(xhttp.Vary)
			if tt.wantVaryHeader && gotVary != xhttp.AcceptEncoding {
				t.Errorf("Vary header = %q, want %q", gotVary, xhttp.AcceptEncoding)
			}
			if !tt.wantVaryHeader && gotVary != "" {
				t.Errorf("Vary header = %q, want empty", gotVary)
			}

			gotEncoding := resp.Header.Get(xhttp.ContentEncoding)
			if tt.wantContentEncoded && gotEncoding != gzipEncoding {
				t.Errorf("Content-Encoding = %q, want %q", gotEncoding, gzipEncoding)
			}
			if !tt.wantContentEncoded && tt.existingEncoding == "" && gotEncoding != "" {
				t.Errorf("Content-Encoding = %q, want empty", gotEncoding)
			}
			if tt.existingEncoding != "" && gotEncoding != tt.existingEncoding {
				t.Errorf("Content-Encoding = %q, want %q (existing)", gotEncoding, tt.existingEncoding)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}

			if tt.wantCompressed {
				decompressed, err := decompressGzip(body)
				if err != nil {
					t.Fatalf("failed to decompress: %v", err)
				}
				if string(decompressed) != tt.responseBody {
					t.Errorf("decompressed body = %q, want %q", string(decompressed), tt.responseBody)
				}
			} else {
				if string(body) != tt.responseBody {
					t.Errorf("body = %q, want %q", string(body), tt.responseBody)
				}
			}
		})
	}
}

func TestGzipFlusher(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", 2000)))
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		_, _ = w.Write([]byte(strings.Repeat("y", 500)))
	})

	wrapped := Gzip(handler)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/test", nil)
	req.Header.Set(xhttp.AcceptEncoding, "gzip")

	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close() //nolint:errcheck

	if resp.Header.Get(xhttp.ContentEncoding) != gzipEncoding {
		t.Error("expected gzip encoding")
	}

	body, _ := io.ReadAll(resp.Body)
	decompressed, err := decompressGzip(body)
	if err != nil {
		t.Fatalf("failed to decompress: %v", err)
	}

	expected := strings.Repeat("x", 2000) + strings.Repeat("y", 500)
	if string(decompressed) != expected {
		t.Errorf("decompressed body length = %d, want %d", len(decompressed), len(expected))
	}
}

func TestGzipUnwrap(t *testing.T) {
	t.Parallel()

	var capturedWriter http.ResponseWriter

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		capturedWriter = w
		_, _ = w.Write([]byte("test"))
	})

	wrapped := Gzip(handler)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/test", nil)
	req.Header.Set(xhttp.AcceptEncoding, "gzip")

	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	gw, ok := capturedWriter.(*gzipResponseWriter)
	if !ok {
		t.Fatal("expected gzipResponseWriter")
	}

	if gw.Unwrap() == nil {
		t.Error("Unwrap() returned nil")
	}
}

func decompressGzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close() //nolint:errcheck

	return io.ReadAll(reader)
}
