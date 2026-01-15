package codex

import (
	"context"
	"strings"
)

type contextKey string

const outputPathKey contextKey = "codex_output_path"

func WithOutputPath(ctx context.Context, path string) context.Context {
	if strings.TrimSpace(path) == "" {
		return ctx
	}
	return context.WithValue(ctx, outputPathKey, path)
}

func outputPathFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if value, ok := ctx.Value(outputPathKey).(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}
