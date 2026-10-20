package services

import "context"

// The session id travels from the /config cookie to the provider selection
// inside the request context; empty string means "no session tuning".
type sessionIDKey struct{}

func WithSessionID(ctx context.Context, sid string) context.Context {
	if sid == "" {
		return ctx
	}
	return context.WithValue(ctx, sessionIDKey{}, sid)
}

func sessionIDFromContext(ctx context.Context) string {
	if sid, ok := ctx.Value(sessionIDKey{}).(string); ok {
		return sid
	}
	return ""
}
