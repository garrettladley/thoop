package commands

import (
	"github.com/garrettladley/thoop/internal/client/whoop"
	"golang.org/x/oauth2"
)

type AuthStatusMsg struct {
	HasToken bool
	Err      error
}

type AuthFlowStartMsg struct{}

type AuthFlowResultMsg struct {
	Token *oauth2.Token
	Err   error
}

type TokenCheckTickMsg struct{}

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
