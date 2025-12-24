package whoop

import "context"

type UserService interface {
	GetProfile(ctx context.Context) (*UserProfile, error)
	GetBodyMeasurement(ctx context.Context) (*BodyMeasurement, error)
	RevokeAccess(ctx context.Context) error
}

type CycleService interface {
	Get(ctx context.Context, id int64) (*Cycle, error)
	List(ctx context.Context, params *ListParams) (*PaginatedResponse[Cycle], error)
	GetSleep(ctx context.Context, cycleID int64) (*Sleep, error)
	GetRecovery(ctx context.Context, cycleID int64) (*Recovery, error)
}

type RecoveryService interface {
	List(ctx context.Context, params *ListParams) (*PaginatedResponse[Recovery], error)
}

type SleepService interface {
	Get(ctx context.Context, id string) (*Sleep, error)
	List(ctx context.Context, params *ListParams) (*PaginatedResponse[Sleep], error)
}

type WorkoutService interface {
	Get(ctx context.Context, id string) (*Workout, error)
	List(ctx context.Context, params *ListParams) (*PaginatedResponse[Workout], error)
}
