package oauth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	sqlitec "github.com/garrettladley/thoop/internal/sqlc/sqlite"
	"github.com/garrettladley/thoop/internal/version"
	"github.com/garrettladley/thoop/internal/xhttp"
	"golang.org/x/oauth2"
)

const (
	callbackPath    = "/callback"
	shutdownTime    = 5 * time.Second
	defaultProxyURL = "https://thoop.fly.dev"
)

type Flow interface {
	Run(ctx context.Context) (*oauth2.Token, error)
}

type tokenResult struct {
	token *oauth2.Token
	err   error
}

type callbackHandler func(w http.ResponseWriter, r *http.Request) (*oauth2.Token, error)

type ServerFlow struct {
	serverURL string
	querier   sqlitec.Querier
}

var _ Flow = (*ServerFlow)(nil)

func NewServerFlow(querier sqlitec.Querier) *ServerFlow {
	return &ServerFlow{
		serverURL: defaultProxyURL,
		querier:   querier,
	}
}

func NewServerFlowWithURL(serverURL string, querier sqlitec.Querier) *ServerFlow {
	return &ServerFlow{
		serverURL: serverURL,
		querier:   querier,
	}
}

func (f *ServerFlow) Run(ctx context.Context) (*oauth2.Token, error) {
	return runFlow(ctx, f.querier, f.authURL, serverCallbackHandler)
}

func (f *ServerFlow) authURL(port string) string {
	return fmt.Sprintf("%s/auth/start?%s=%s&%s=%s",
		f.serverURL,
		ParamLocalPort, port,
		ParamClientVersion, url.QueryEscape(version.Get()))
}

type DirectFlow struct {
	config  *oauth2.Config
	querier sqlitec.Querier
	state   string
}

var _ Flow = (*DirectFlow)(nil)

func NewDirectFlow(config *oauth2.Config, querier sqlitec.Querier) (*DirectFlow, error) {
	state, err := GenerateState()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}
	return &DirectFlow{
		config:  config,
		querier: querier,
		state:   state,
	}, nil
}

func (f *DirectFlow) Run(ctx context.Context) (*oauth2.Token, error) {
	return runFlow(ctx, f.querier, f.authURL, f.callbackHandler())
}

func (f *DirectFlow) authURL(_ string) string {
	return f.config.AuthCodeURL(f.state, oauth2.AccessTypeOffline)
}

func (f *DirectFlow) callbackHandler() callbackHandler {
	return func(w http.ResponseWriter, r *http.Request) (*oauth2.Token, error) {
		if !ValidateState(f.state, r.URL.Query().Get("state")) {
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			return nil, errors.New("invalid state parameter")
		}

		if errParam := r.URL.Query().Get("error"); errParam != "" {
			errDesc := r.URL.Query().Get("error_description")
			http.Error(w, fmt.Sprintf("OAuth error: %s", errDesc), http.StatusBadRequest)
			return nil, fmt.Errorf("oauth error: %s - %s", errParam, errDesc)
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Missing authorization code", http.StatusBadRequest)
			return nil, errors.New("missing authorization code")
		}

		token, err := f.config.Exchange(r.Context(), code)
		if err != nil {
			http.Error(w, "Failed to exchange authorization code", http.StatusInternalServerError)
			return nil, fmt.Errorf("failed to exchange code: %w", err)
		}

		return token, nil
	}
}

func runFlow(
	ctx context.Context,
	querier sqlitec.Querier,
	authURL func(port string) string,
	handler callbackHandler,
) (*oauth2.Token, error) {
	resultCh := make(chan tokenResult, 1)

	server, port, err := startCallbackServer(handler, resultCh)
	if err != nil {
		return nil, fmt.Errorf("failed to start callback server: %w", err)
	}

	url := authURL(port)

	fmt.Printf("Opening browser for authorization...\n")
	fmt.Printf("If the browser doesn't open, visit:\n%s\n\n", url)

	if err := openBrowser(url); err != nil {
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

		if err := saveToken(ctx, querier, result.token); err != nil {
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

func startCallbackServer(handler callbackHandler, resultCh chan<- tokenResult) (*http.Server, string, error) {
	mux := http.NewServeMux()

	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		token, err := handler(w, r)
		if err != nil {
			resultCh <- tokenResult{err: err}
			return
		}
		writeSuccessHTML(w)
		resultCh <- tokenResult{token: token}
	})

	listener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", "0"))
	if err != nil {
		return nil, "", fmt.Errorf("failed to start listener: %w", err)
	}

	_, port, _ := net.SplitHostPort(listener.Addr().String())

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			resultCh <- tokenResult{err: fmt.Errorf("server error: %w", err)}
		}
	}()

	return server, port, nil
}

func serverCallbackHandler(w http.ResponseWriter, r *http.Request) (*oauth2.Token, error) {
	if errParam := r.URL.Query().Get(ParamError); errParam != "" {
		errDesc := r.URL.Query().Get(ParamErrorDescription)

		if ErrorCode(errParam) == ErrorCodeIncompatibleVersion {
			minVersion := r.URL.Query().Get(ParamMinVersion)
			writeVersionErrorHTML(w, errDesc, minVersion)
			fmt.Fprintf(os.Stderr, "\nVersion incompatibility: %s\n", errDesc)
			fmt.Fprintf(os.Stderr, "Please upgrade: go install github.com/garrettladley/thoop/cmd/thoop@latest\n\n")
			return nil, fmt.Errorf("version incompatibility: %s", errDesc)
		}

		http.Error(w, fmt.Sprintf("OAuth error: %s", errDesc), http.StatusBadRequest)
		return nil, fmt.Errorf("oauth error: %s - %s", errParam, errDesc)
	}

	accessToken := r.URL.Query().Get("access_token")
	if accessToken == "" {
		http.Error(w, "Missing access token", http.StatusBadRequest)
		return nil, errors.New("missing access_token")
	}

	tokenType := r.URL.Query().Get("token_type")
	if tokenType == "" {
		tokenType = "Bearer"
	}

	var expiry time.Time
	if expiresInStr := r.URL.Query().Get("expires_in"); expiresInStr != "" {
		if expiresIn, err := strconv.Atoi(expiresInStr); err == nil {
			expiry = time.Now().Add(time.Duration(expiresIn) * time.Second)
		}
	}

	return &oauth2.Token{
		AccessToken:  accessToken,
		TokenType:    tokenType,
		RefreshToken: r.URL.Query().Get("refresh_token"),
		Expiry:       expiry,
	}, nil
}

func writeVersionErrorHTML(w http.ResponseWriter, errDesc string, minVersion string) {
	xhttp.SetHeaderContentTypeTextHTML(w)
	upgradeCmd := "go install github.com/garrettladley/thoop/cmd/thoop@latest"
	_, _ = fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>Version Incompatibility</title></head>
<body>
<h1>Version Incompatibility</h1>
<p>%s</p>
<p>Please upgrade to v%s or later:</p>
<pre>%s</pre>
<p>Then return to the terminal and try again.</p>
</body>
</html>`, errDesc, minVersion, upgradeCmd)
}

func saveToken(ctx context.Context, querier sqlitec.Querier, token *oauth2.Token) error {
	params := sqlitec.UpsertTokenParams{
		AccessToken: token.AccessToken,
		TokenType:   token.TokenType,
		Expiry:      token.Expiry,
	}

	if token.RefreshToken != "" {
		params.RefreshToken = &token.RefreshToken
	}

	return querier.UpsertToken(ctx, params)
}

func writeSuccessHTML(w http.ResponseWriter) {
	xhttp.SetHeaderContentTypeTextHTML(w)
	_, _ = fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head><title>Authorization Successful</title></head>
<body>
<h1>Authorization Successful</h1>
<p>You can close this window and return to the terminal.</p>
</body>
</html>`)
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
