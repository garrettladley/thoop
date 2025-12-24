package whoop

type ScoreState string

const (
	ScoreStateScored       ScoreState = "SCORED"
	ScoreStatePendingScore ScoreState = "PENDING_SCORE"
	ScoreStateUnscorable   ScoreState = "UNSCORABLE"
)
