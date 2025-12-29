package commands

import (
	"github.com/garrettladley/thoop/internal/client/whoop"
	"golang.org/x/oauth2"
)

type AuthStatusMsg struct {
	HasToken bool
	Err      error
}

// AuthFlowStartMsg signals that the auth flow should start
type AuthFlowStartMsg struct{}

// AuthFlowResultMsg is the result of the OAuth flow
type AuthFlowResultMsg struct {
	Token *oauth2.Token
	Err   error
}

// TokenCheckTickMsg is sent periodically to check token expiry
type TokenCheckTickMsg struct{}

// TokenRefreshResultMsg is the result of a token refresh attempt
type TokenRefreshResultMsg struct {
	Refreshed bool
	Err       error
}

type CycleMsg struct {
	Cycle *whoop.Cycle
	Err   error
}

type SleepMsg struct {
	Sleep *whoop.Sleep
	Err   error
}

type RecoveryMsg struct {
	Recovery *whoop.Recovery
	Err      error
}
