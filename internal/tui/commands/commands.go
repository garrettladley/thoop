package commands

import "github.com/garrettladley/thoop/internal/client/whoop"

type AuthStatusMsg struct {
	HasToken bool
	Err      error
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
