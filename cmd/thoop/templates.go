//go:build dev

package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/garrettladley/thoop/internal/oauth/templates"
	"github.com/garrettladley/thoop/internal/xhttp"
	"github.com/garrettladley/thoop/internal/xtempl"
	"github.com/spf13/cobra"
)

func templatesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "templates",
		Short: "Preview OAuth templates",
		Long:  "Starts a local server at localhost:8080 to preview OAuth flow templates.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTemplatesServer(cmd.Context())
		},
	}
}

func runTemplatesServer(ctx context.Context) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		xhttp.SetHeaderContentTypeTextHTMLCharsetUTF8(w)
		_, _ = fmt.Fprint(w, landingHTML)
	})

	mux.HandleFunc("/success", func(w http.ResponseWriter, r *http.Request) {
		_ = xtempl.Render(w, r, templates.Success())
	})

	mux.HandleFunc("/version-error", func(w http.ResponseWriter, r *http.Request) {
		_ = xtempl.Render(w, r, templates.VersionError("Your client version is too old"))
	})

	mux.HandleFunc("/account-banned", func(w http.ResponseWriter, r *http.Request) {
		_ = xtempl.Render(w, r, templates.AccountBanned())
	})

	mux.HandleFunc("/rate-limited", func(w http.ResponseWriter, r *http.Request) {
		_ = xtempl.Render(w, r, templates.RateLimited())
	})

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}

	addr := listener.Addr().String()
	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	fmt.Printf("Template preview server running at http://%s\n", addr)
	fmt.Println("Press Ctrl+C to stop")

	if err := openBrowser(ctx, "http://"+addr); err != nil {
		fmt.Printf("Failed to open browser: %v\n", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

func openBrowser(ctx context.Context, url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "open", url)
	case "linux":
		cmd = exec.CommandContext(ctx, "xdg-open", url)
	case "windows":
		cmd = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}

const landingHTML = `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Template Previews</title>
	<style>
		body {
			font-family: system-ui, sans-serif;
			max-width: 600px;
			margin: 2rem auto;
			padding: 1rem;
			background: #101518;
			color: #e4e4e7;
		}
		a {
			color: #00F19F;
			display: block;
			padding: 0.5rem 0;
		}
		h1 { color: #fff; }
	</style>
</head>
<body>
	<h1>OAuth Template Previews</h1>
	<a href="/success">Success</a>
	<a href="/version-error">Version Error</a>
	<a href="/account-banned">Account Banned</a>
	<a href="/rate-limited">Rate Limited</a>
</body>
</html>`
