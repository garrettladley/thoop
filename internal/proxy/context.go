package proxy

import "context"

type requestIDKey struct{}

func SetRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

func GetRequestID(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(requestIDKey{}).(string)
	return requestID, ok
}

type whoopUserIDKey struct{}

func SetWhoopUserID(ctx context.Context, whoopUserID string) context.Context {
	return context.WithValue(ctx, whoopUserIDKey{}, whoopUserID)
}

func GetWhoopUserID(ctx context.Context) (string, bool) {
	whoopUserID, ok := ctx.Value(whoopUserIDKey{}).(string)
	return whoopUserID, ok
}
