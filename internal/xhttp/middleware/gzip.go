package middleware

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/garrettladley/thoop/internal/xhttp"
)

const (
	gzipMinSize   = 1024 // 1KB minimum before compression kicks in
	gzipEncoding  = "gzip"
	sseStreamPath = "/api/notifications/stream"
)

var gzipWriterPool = sync.Pool{
	New: func() any {
		return gzip.NewWriter(nil)
	},
}

type gzipResponseWriter struct {
	http.ResponseWriter
	writer      *gzip.Writer
	buf         bytes.Buffer
	wroteHeader bool
	statusCode  int
	useGzip     bool
	decided     bool
}

var (
	_ http.ResponseWriter = (*gzipResponseWriter)(nil)
	_ http.Flusher        = (*gzipResponseWriter)(nil)
	_ io.Closer           = (*gzipResponseWriter)(nil)
)

func (g *gzipResponseWriter) WriteHeader(code int) {
	g.statusCode = code
	g.wroteHeader = true
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	if !g.wroteHeader {
		g.WriteHeader(http.StatusOK)
	}

	if g.decided {
		if g.useGzip {
			n, err := g.writer.Write(b)
			if err != nil {
				return n, fmt.Errorf("failed to write gzip: %w", err)
			}
			return n, nil
		}
		n, err := g.ResponseWriter.Write(b)
		if err != nil {
			return n, fmt.Errorf("failed to write response: %w", err)
		}
		return n, nil
	}

	g.buf.Write(b)

	if g.buf.Len() >= gzipMinSize {
		g.decided = true
		if g.ResponseWriter.Header().Get(xhttp.ContentEncoding) == "" {
			g.useGzip = true
			return g.startGzip()
		}
		return g.flushUncompressed()
	}

	return len(b), nil
}

func (g *gzipResponseWriter) startGzip() (int, error) {
	g.ResponseWriter.Header().Set(xhttp.ContentEncoding, gzipEncoding)
	g.ResponseWriter.Header().Del(xhttp.ContentLength)
	g.ResponseWriter.WriteHeader(g.statusCode)

	g.writer = gzipWriterPool.Get().(*gzip.Writer)
	g.writer.Reset(g.ResponseWriter)

	n, err := g.writer.Write(g.buf.Bytes())
	if err != nil {
		return n, fmt.Errorf("failed to write to gzip writer: %w", err)
	}
	return n, nil
}

func (g *gzipResponseWriter) flushUncompressed() (int, error) {
	g.ResponseWriter.WriteHeader(g.statusCode)
	n, err := g.ResponseWriter.Write(g.buf.Bytes())
	if err != nil {
		return n, fmt.Errorf("failed to write uncompressed response: %w", err)
	}
	return n, nil
}

func (g *gzipResponseWriter) Close() error {
	if !g.decided {
		g.decided = true
		g.ResponseWriter.WriteHeader(g.statusCode)
		_, err := g.ResponseWriter.Write(g.buf.Bytes())
		if err != nil {
			return fmt.Errorf("failed to write buffered response: %w", err)
		}
		return nil
	}

	if g.useGzip && g.writer != nil {
		err := g.writer.Close()
		gzipWriterPool.Put(g.writer)
		g.writer = nil
		if err != nil {
			return fmt.Errorf("failed to close gzip writer: %w", err)
		}
	}

	return nil
}

func (g *gzipResponseWriter) Flush() {
	if g.useGzip && g.writer != nil {
		_ = g.writer.Flush()
	}
	if flusher, ok := g.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (g *gzipResponseWriter) Unwrap() http.ResponseWriter {
	return g.ResponseWriter
}

func Gzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !clientAcceptsGzip(r) || isExcludedPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set(xhttp.Vary, xhttp.AcceptEncoding)

		gw := &gzipResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}
		defer gw.Close() //nolint:errcheck // best-effort flush on response completion

		next.ServeHTTP(gw, r)
	})
}

func clientAcceptsGzip(r *http.Request) bool {
	return strings.Contains(r.Header.Get(xhttp.AcceptEncoding), gzipEncoding)
}

func isExcludedPath(path string) bool {
	return path == sseStreamPath
}
