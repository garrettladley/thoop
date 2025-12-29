package xcontext

import "context"

type sessionIDKey struct{}

func SetSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionIDKey{}, sessionID)
}

func GetSessionID(ctx context.Context) (string, bool) {
	sessionID, ok := ctx.Value(sessionIDKey{}).(string)
	return sessionID, ok
}
