package xcontext

import "context"

type whoopUserIDKey struct{}

func SetWhoopUserID(ctx context.Context, whoopUserID string) context.Context {
	return context.WithValue(ctx, whoopUserIDKey{}, whoopUserID)
}

func GetWhoopUserID(ctx context.Context) (string, bool) {
	whoopUserID, ok := ctx.Value(whoopUserIDKey{}).(string)
	return whoopUserID, ok
}
