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

	"github.com/garrettladley/thoop"
	"github.com/garrettladley/thoop/internal/keyring"
	"github.com/garrettladley/thoop/internal/oauth/templates"
	"github.com/garrettladley/thoop/internal/sqlite/sqlc"
	"github.com/garrettladley/thoop/internal/xtempl"
	"golang.org/x/oauth2"
)

const (
	callbackPath     = "/callback"
	shutdownTime     = 5 * time.Second
	defaultServerURL = "https://thoop.fly.dev"
	defaultTokenType = "Bearer"
)

var (
	ErrInvalidState       = errors.New("invalid state parameter")
	ErrMissingAuthCode    = errors.New("missing authorization code")
	ErrMissingAccessToken = errors.New("missing access_token")
)

type AuthResult struct {
	Token  *oauth2.Token
	APIKey string
}

type Flow interface {
	Run(ctx context.Context) (*AuthResult, error)
}

type tokenResult struct {
	token  *oauth2.Token
	apiKey string
	err    error
}

type callbackHandler func(w http.ResponseWriter, r *http.Request) (*oauth2.Token, string, error)

type ServerFlow struct {
	serverURL string
	querier   litesqlc.Querier
	keyring   keyring.Store
}

var _ Flow = (*ServerFlow)(nil)

func NewServerFlow(serverURL string, querier litesqlc.Querier, kr keyring.Store) *ServerFlow {
	return &ServerFlow{
		serverURL: serverURL,
		querier:   querier,
		keyring:   kr,
	}
}

func (f *ServerFlow) Run(ctx context.Context) (*AuthResult, error) {
	return runFlow(ctx, f.querier, f.keyring, f.authURL, serverCallbackHandler)
}

func (f *ServerFlow) authURL(port string) string {
	return fmt.Sprintf("%s/auth/start?%s=%s&%s=%s",
		f.serverURL,
		ParamLocalPort, port,
		ParamClientVersion, url.QueryEscape(thoop.Version))
}

type DirectFlow struct {
	config  *oauth2.Config
	querier litesqlc.Querier
	keyring keyring.Store
	state   string
}

var _ Flow = (*DirectFlow)(nil)

func NewDirectFlow(config *oauth2.Config, querier litesqlc.Querier, kr keyring.Store) (*DirectFlow, error) {
	state, err := GenerateState()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}
	return &DirectFlow{
		config:  config,
		querier: querier,
		keyring: kr,
		state:   state,
	}, nil
}

func (f *DirectFlow) Run(ctx context.Context) (*AuthResult, error) {
	return runFlow(ctx, f.querier, f.keyring, f.authURL, f.callbackHandler())
}

func (f *DirectFlow) authURL(_ string) string {
	return f.config.AuthCodeURL(f.state, oauth2.AccessTypeOffline)
}

func (f *DirectFlow) callbackHandler() callbackHandler {
	return func(w http.ResponseWriter, r *http.Request) (*oauth2.Token, string, error) {
		if !ValidateState(f.state, r.URL.Query().Get("state")) {
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			return nil, "", ErrInvalidState
		}

		if errParam := r.URL.Query().Get("error"); errParam != "" {
			errDesc := r.URL.Query().Get("error_description")
			http.Error(w, fmt.Sprintf("OAuth error: %s", errDesc), http.StatusBadRequest)
			return nil, "", fmt.Errorf("oauth error: %s - %s", errParam, errDesc)
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Missing authorization code", http.StatusBadRequest)
			return nil, "", ErrMissingAuthCode
		}

		token, err := f.config.Exchange(r.Context(), code)
		if err != nil {
			http.Error(w, "Failed to exchange authorization code", http.StatusInternalServerError)
			return nil, "", fmt.Errorf("failed to exchange code: %w", err)
		}

		return token, "", nil
	}
}

func runFlow(
	ctx context.Context,
	querier litesqlc.Querier,
	kr keyring.Store,
	authURL func(port string) string,
	handler callbackHandler,
) (*AuthResult, error) {
	resultCh := make(chan tokenResult, 1)

	server, port, err := startCallbackServer(ctx, handler, resultCh)
	if err != nil {
		return nil, fmt.Errorf("failed to start callback server: %w", err)
	}

	url := authURL(port)

	fmt.Printf("Opening browser for authorization...\n")
	fmt.Printf("If the browser doesn't open, visit:\n%s\n\n", url)

	if err := openBrowser(ctx, url); err != nil {
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

		if err := saveToken(ctx, querier, kr, result.token, result.apiKey); err != nil {
			return nil, fmt.Errorf("failed to save token: %w", err)
		}

		return &AuthResult{Token: result.token, APIKey: result.apiKey}, nil

	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTime)
		defer cancel()

		_ = server.Shutdown(shutdownCtx)

		return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
	}
}

func startCallbackServer(ctx context.Context, handler callbackHandler, resultCh chan<- tokenResult) (*http.Server, string, error) {
	mux := http.NewServeMux()

	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		token, apiKey, err := handler(w, r)
		if err != nil {
			resultCh <- tokenResult{err: err}
			return
		}
		_ = xtempl.Render(w, r, templates.Success())
		resultCh <- tokenResult{token: token, apiKey: apiKey}
	})

	lc := &net.ListenConfig{}
	listener, err := lc.Listen(ctx, "tcp", net.JoinHostPort("127.0.0.1", "0"))
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

func serverCallbackHandler(w http.ResponseWriter, r *http.Request) (*oauth2.Token, string, error) {
	if errParam := r.URL.Query().Get(ParamError); errParam != "" {
		errDesc := r.URL.Query().Get(ParamErrorDescription)

		if ErrorCode(errParam) == ErrorCodeIncompatibleVersion {
			_ = xtempl.Render(w, r, templates.VersionError(errDesc))
			fmt.Fprintf(os.Stderr, "\nVersion incompatibility: %s\n", errDesc)
			fmt.Fprintf(os.Stderr, "Please upgrade: thoop upgrade\n\n")
			return nil, "", fmt.Errorf("version incompatibility: %s", errDesc)
		}

		if ErrorCode(errParam) == ErrorCodeAccountBanned {
			_ = xtempl.Render(w, r, templates.AccountBanned())
			fmt.Fprintf(os.Stderr, "\nAccount banned: %s\n", errDesc)
			return nil, "", fmt.Errorf("account banned: %s", errDesc)
		}

		if ErrorCode(errParam) == ErrorCodeRateLimited {
			_ = xtempl.Render(w, r, templates.RateLimited())
			fmt.Fprintf(os.Stderr, "\nRate limited: %s\n", errDesc)
			return nil, "", fmt.Errorf("rate limited: %s", errDesc)
		}

		http.Error(w, fmt.Sprintf("OAuth error: %s", errDesc), http.StatusBadRequest)
		return nil, "", fmt.Errorf("oauth error: %s - %s", errParam, errDesc)
	}

	accessToken := r.URL.Query().Get("access_token")
	if accessToken == "" {
		http.Error(w, "Missing access token", http.StatusBadRequest)
		return nil, "", ErrMissingAccessToken
	}

	tokenType := r.URL.Query().Get("token_type")
	if tokenType == "" {
		tokenType = defaultTokenType
	}

	var expiry time.Time
	if expiresInStr := r.URL.Query().Get("expires_in"); expiresInStr != "" {
		if expiresIn, err := strconv.Atoi(expiresInStr); err == nil {
			expiry = time.Now().Add(time.Duration(expiresIn) * time.Second)
		}
	}

	apiKey := r.URL.Query().Get("api_key")

	return &oauth2.Token{
		AccessToken:  accessToken,
		TokenType:    tokenType,
		RefreshToken: r.URL.Query().Get("refresh_token"),
		Expiry:       expiry,
	}, apiKey, nil
}

func saveToken(ctx context.Context, querier litesqlc.Querier, kr keyring.Store, token *oauth2.Token, apiKey string) error {
	if err := kr.Set(keyring.KeyAccessToken, token.AccessToken); err != nil {
		return fmt.Errorf("failed to save access token to keyring: %w", err)
	}

	if token.RefreshToken != "" {
		if err := kr.Set(keyring.KeyRefreshToken, token.RefreshToken); err != nil {
			return fmt.Errorf("failed to save refresh token to keyring: %w", err)
		}
	}

	if apiKey != "" {
		if err := kr.Set(keyring.KeyAPIKey, apiKey); err != nil {
			return fmt.Errorf("failed to save API key to keyring: %w", err)
		}
	}

	tokenType := token.TokenType
	if tokenType == "" {
		tokenType = defaultTokenType
	}

	params := litesqlc.UpsertTokenMetadataParams{
		TokenType: tokenType,
		Expiry:    token.Expiry,
	}

	if err := querier.UpsertTokenMetadata(ctx, params); err != nil {
		return fmt.Errorf("failed to save token metadata: %w", err)
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

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start browser: %w", err)
	}
	return nil
}
