package auth

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/garrettladley/thoop/internal/client/whoop"
	intoauth "github.com/garrettladley/thoop/internal/oauth"
	"github.com/garrettladley/thoop/internal/service/user"
	"github.com/garrettladley/thoop/internal/storage"
	"github.com/garrettladley/thoop/internal/version"
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
		return nil, err
	}

	entry := storage.StateEntry{
		LocalPort:     req.LocalPort,
		ClientVersion: clientVersion,
		CreatedAt:     time.Now(),
	}

	if err := s.stateStore.Set(ctx, state, entry, stateTTL); err != nil {
		return nil, err
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
		return nil, err
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
		return nil, err
	}

	rlState, err := s.whoopLimiter.CheckAndIncrement(ctx, "oauth")
	if err != nil {
		return nil, err
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
		return nil, err
	}

	apiKey, banned, err := s.userService.GetOrCreateUser(ctx, profile.UserID)
	if err != nil {
		return nil, err
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
