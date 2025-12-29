package commands

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/garrettladley/thoop/internal/oauth"
)

func CheckAuthCmd(ctx context.Context, checker oauth.TokenChecker) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		hasToken, err := checker.HasToken(ctx)
		return AuthStatusMsg{HasToken: hasToken, Err: err}
	}
}

// StartAuthFlowCmd runs the OAuth flow and returns the result
func StartAuthFlowCmd(ctx context.Context, flow oauth.Flow) tea.Cmd {
	return func() tea.Msg {
		token, err := flow.Run(ctx)
		return AuthFlowResultMsg{Token: token, Err: err}
	}
}

// TokenCheckTickCmd returns a tick command for periodic token checks
func TokenCheckTickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return TokenCheckTickMsg{}
	})
}

// RefreshTokenIfNeededCmd checks if the token needs refresh and refreshes it
func RefreshTokenIfNeededCmd(ctx context.Context, tokenSource *oauth.DBTokenSource, threshold time.Duration) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		token, err := tokenSource.RefreshIfNeeded(ctx, threshold)
		if err != nil {
			return TokenRefreshResultMsg{Refreshed: false, Err: err}
		}

		return TokenRefreshResultMsg{Refreshed: token != nil, Err: nil}
	}
}
