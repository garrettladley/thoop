package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	go_json "github.com/goccy/go-json"

	"github.com/garrettladley/thoop/internal/client/whoop"
	intoauth "github.com/garrettladley/thoop/internal/oauth"
	"github.com/garrettladley/thoop/internal/service/user"
	"github.com/garrettladley/thoop/internal/storage"
	"github.com/garrettladley/thoop/internal/version"
	"github.com/garrettladley/thoop/internal/xslog"
	"golang.org/x/oauth2"
)

const stateTTL = 5 * time.Minute

type OAuth struct {
	config       *oauth2.Config
	stateStore   storage.StateStore
	userService  user.Service
	whoopLimiter storage.WhoopRateLimiter
}

var _ Service = (*OAuth)(nil)

func NewOAuth(
	oauthConfig *oauth2.Config,
	stateStore storage.StateStore,
	userService user.Service,
	whoopLimiter storage.WhoopRateLimiter,
) *OAuth {
	return &OAuth{
		config:       oauthConfig,
		stateStore:   stateStore,
		userService:  userService,
		whoopLimiter: whoopLimiter,
	}
}

func (s *OAuth) StartAuth(ctx context.Context, req StartAuthRequest) (*StartAuthResult, error) {
	if !isValidPort(req.LocalPort) {
		return nil, ErrInvalidPort
	}

	clientVersion := req.ClientVersion
	if clientVersion == "" {
		clientVersion = "unknown"
	}

	if verr := version.CheckCompatibility(clientVersion); verr != nil {
		return nil, &VersionError{MinVersion: verr.MinVersion}
	}

	state, err := intoauth.GenerateState()
	if err != nil {
		return nil, fmt.Errorf("generating state: %w", err)
	}

	entry := storage.StateEntry{
		LocalPort:     req.LocalPort,
		ClientVersion: clientVersion,
		CreatedAt:     time.Now(),
	}

	if err := s.stateStore.Set(ctx, state, entry, stateTTL); err != nil {
		return nil, fmt.Errorf("storing state: %w", err)
	}

	authURL := s.config.AuthCodeURL(state, oauth2.AccessTypeOffline)

	return &StartAuthResult{AuthURL: authURL}, nil
}

func (s *OAuth) HandleCallback(ctx context.Context, req CallbackRequest) (*CallbackResult, error) {
	if req.State == "" {
		return nil, ErrInvalidState
	}

	entry, err := s.stateStore.GetAndDelete(ctx, req.State)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, ErrInvalidState
	}
	if err != nil {
		return nil, fmt.Errorf("retrieving state: %w", err)
	}

	if req.ErrorCode != "" {
		return nil, &AuthError{
			Err:       ErrAuthDenied,
			LocalPort: entry.LocalPort,
			ErrorCode: req.ErrorCode,
			ErrorDesc: req.ErrorDesc,
		}
	}

	if req.Code == "" {
		return nil, &AuthError{
			Err:       ErrInvalidState,
			LocalPort: entry.LocalPort,
			ErrorCode: "invalid_request",
			ErrorDesc: "missing authorization code",
		}
	}

	exchangeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	token, err := s.config.Exchange(exchangeCtx, req.Code)
	if err != nil {
		return nil, fmt.Errorf("exchanging code for token: %w", err)
	}

	rlState, err := s.whoopLimiter.CheckAndIncrement(ctx, "oauth")
	if err != nil {
		return nil, fmt.Errorf("checking rate limit: %w", err)
	}
	if !rlState.Allowed {
		return nil, &AuthError{
			Err:       ErrRateLimited,
			LocalPort: entry.LocalPort,
			ErrorCode: string(intoauth.ErrorCodeRateLimited),
			ErrorDesc: "rate limit exceeded, please try again later",
		}
	}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token.AccessToken})
	whoopClient := whoop.New(tokenSource)

	profile, err := whoopClient.User.GetProfile(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting whoop profile: %w", err)
	}

	apiKey, banned, err := s.userService.GetOrCreateUser(ctx, profile.UserID)
	if err != nil {
		return nil, fmt.Errorf("getting or creating user: %w", err)
	}

	if banned {
		return nil, &AuthError{
			Err:       ErrAccountBanned,
			LocalPort: entry.LocalPort,
			ErrorCode: string(intoauth.ErrorCodeAccountBanned),
			ErrorDesc: "your account has been banned",
		}
	}

	return &CallbackResult{
		Token:     token,
		APIKey:    apiKey,
		LocalPort: entry.LocalPort,
	}, nil
}

type tokenResp struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

func (s *OAuth) RefreshToken(ctx context.Context, req RefreshRequest) (*RefreshResult, error) {
	logger := xslog.FromContext(ctx)

	if req.RefreshToken == "" {
		return nil, ErrInvalidRefreshToken
	}

	refreshCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// WHOOP requires scope=offline in the refresh request, which the standard
	// oauth2.TokenSource doesn't include. we _must_ make a manual request.
	// see: https://developer.whoop.com/docs/developing/oauth#receiving-a-refresh-token
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {req.RefreshToken},
		"client_id":     {s.config.ClientID},
		"client_secret": {s.config.ClientSecret},
		"scope":         {"offline"},
	}

	httpReq, err := http.NewRequestWithContext(refreshCtx, http.MethodPost, s.config.Endpoint.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating refresh request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		logger.ErrorContext(ctx, "whoop token refresh request failed", xslog.Error(err))
		return nil, ErrRefreshFailed
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logger.ErrorContext(ctx, "whoop token refresh failed",
			xslog.HTTPStatus(resp.StatusCode),
			xslog.Body(string(body)))
		return nil, ErrRefreshFailed
	}

	var tokenResp tokenResp
	if err := go_json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		logger.ErrorContext(ctx, "failed to decode token response", xslog.Error(err))
		return nil, ErrRefreshFailed
	}

	newToken := &oauth2.Token{
		AccessToken:  tokenResp.AccessToken,
		TokenType:    tokenResp.TokenType,
		RefreshToken: tokenResp.RefreshToken,
		Expiry:       time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}

	logger.InfoContext(
		ctx,
		"whoop token refresh succeeded",
		xslog.Duration(time.Duration(tokenResp.ExpiresIn)*time.Second),
	)

	return &RefreshResult{Token: newToken}, nil
}

func isValidPort(s string) bool {
	if s == "" {
		return false
	}
	port, err := strconv.Atoi(s)
	if err != nil {
		return false
	}
	return port >= 1 && port <= 65535
}
