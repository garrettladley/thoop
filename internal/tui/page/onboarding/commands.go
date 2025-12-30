package onboarding

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	"golang.org/x/oauth2"

	"github.com/garrettladley/thoop/internal/oauth"
)

type AuthStatusMsg struct {
	HasToken bool
	Err      error
}

type AuthFlowResultMsg struct {
	Token *oauth2.Token
	Err   error
}

type TokenCheckTickMsg struct{}

type TokenRefreshResultMsg struct {
	Refreshed bool
	Err       error
}

func CheckAuthCmd(ctx context.Context, checker oauth.TokenChecker) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		hasToken, err := checker.HasToken(ctx)
		return AuthStatusMsg{HasToken: hasToken, Err: err}
	}
}

func StartAuthFlowCmd(ctx context.Context, flow oauth.Flow) tea.Cmd {
	return func() tea.Msg {
		token, err := flow.Run(ctx)
		return AuthFlowResultMsg{Token: token, Err: err}
	}
}

func TokenCheckTickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return TokenCheckTickMsg{}
	})
}

func RefreshTokenIfNeededCmd(ctx context.Context, tokenSource oauth.TokenSource, threshold time.Duration) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		token, err := tokenSource.RefreshIfNeeded(ctx, threshold)
		if err != nil {
			return TokenRefreshResultMsg{Refreshed: false, Err: err}
		}

		return TokenRefreshResultMsg{Refreshed: token != nil, Err: nil}
	}
}
