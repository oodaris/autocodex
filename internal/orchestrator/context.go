package orchestrator

import (
	"context"
	"strings"
)

type contextKey string

const beadIDKey contextKey = "autocodex_bead_id"

func WithBeadID(ctx context.Context, beadID string) context.Context {
	if strings.TrimSpace(beadID) == "" {
		return ctx
	}
	return context.WithValue(ctx, beadIDKey, beadID)
}

func beadIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if value, ok := ctx.Value(beadIDKey).(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}
