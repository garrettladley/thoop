package oauth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/garrettladley/thoop/internal/sqlc"
	"golang.org/x/oauth2"
)

const (
	callbackPath = "/callback"
	shutdownTime = 5 * time.Second
)

type Flow struct {
	config  *oauth2.Config
	querier sqlc.Querier
}

func NewFlow(config *oauth2.Config, querier sqlc.Querier) *Flow {
	return &Flow{
		config:  config,
		querier: querier,
	}
}

func (f *Flow) Run(ctx context.Context) (*oauth2.Token, error) {
	state, err := GenerateState()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	resultCh := make(chan tokenResult, 1)

	server, err := f.startCallbackServer(ctx, state, resultCh)
	if err != nil {
		return nil, fmt.Errorf("failed to start callback server: %w", err)
	}

	authURL := f.config.AuthCodeURL(state, oauth2.AccessTypeOffline)

	fmt.Printf("Opening browser for authorization...\n")
	fmt.Printf("If the browser doesn't open, visit:\n%s\n\n", authURL)

	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Failed to open browser: %v\n", err)
	}

	select {
	case result := <-resultCh:
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTime)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			fmt.Printf("Warning: failed to shutdown server: %v\n", err)
		}

		if result.err != nil {
			return nil, result.err
		}

		if err := f.saveToken(ctx, result.token); err != nil {
			return nil, fmt.Errorf("failed to save token: %w", err)
		}

		return result.token, nil

	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTime)
		defer cancel()

		_ = server.Shutdown(shutdownCtx)

		return nil, ctx.Err()
	}
}

type tokenResult struct {
	token *oauth2.Token
	err   error
}

func (f *Flow) startCallbackServer(ctx context.Context, expectedState string, resultCh chan<- tokenResult) (*http.Server, error) {
	mux := http.NewServeMux()

	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		state := r.URL.Query().Get("state")
		if !ValidateState(expectedState, state) {
			resultCh <- tokenResult{err: errors.New("invalid state parameter")}
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			return
		}

		if errParam := r.URL.Query().Get("error"); errParam != "" {
			errDesc := r.URL.Query().Get("error_description")
			resultCh <- tokenResult{err: fmt.Errorf("oauth error: %s - %s", errParam, errDesc)}
			http.Error(w, fmt.Sprintf("OAuth error: %s", errDesc), http.StatusBadRequest)

			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			resultCh <- tokenResult{err: errors.New("missing authorization code")}
			http.Error(w, "Missing authorization code", http.StatusBadRequest)

			return
		}

		token, err := f.config.Exchange(ctx, code)
		if err != nil {
			resultCh <- tokenResult{err: fmt.Errorf("failed to exchange code: %w", err)}
			http.Error(w, "Failed to exchange authorization code", http.StatusInternalServerError)

			return
		}

		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head><title>Authorization Successful</title></head>
<body>
<h1>Authorization Successful</h1>
<p>You can close this window and return to the terminal.</p>
</body>
</html>`)

		resultCh <- tokenResult{token: token}
	})

	addr := net.JoinHostPort("127.0.0.1", "8080")
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			resultCh <- tokenResult{err: fmt.Errorf("server error: %w", err)}
		}
	}()

	return server, nil
}

func (f *Flow) saveToken(ctx context.Context, token *oauth2.Token) error {
	params := sqlc.UpsertTokenParams{
		AccessToken: token.AccessToken,
		TokenType:   token.TokenType,
		Expiry:      token.Expiry,
	}

	if token.RefreshToken != "" {
		params.RefreshToken = &token.RefreshToken
	}

	return f.querier.UpsertToken(ctx, params)
}

func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}
