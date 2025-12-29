package xcontext

import "context"

type whoopUserIDKey struct{}

func SetWhoopUserID(ctx context.Context, whoopUserID int64) context.Context {
	return context.WithValue(ctx, whoopUserIDKey{}, whoopUserID)
}

func GetWhoopUserID(ctx context.Context) (int64, bool) {
	whoopUserID, ok := ctx.Value(whoopUserIDKey{}).(int64)
	return whoopUserID, ok
}
