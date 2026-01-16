package codex

import (
	"context"
	"io"
	"strings"
	"time"
)

type contextKey string

const outputPathKey contextKey = "codex_output_path"
const pidReporterKey contextKey = "codex_pid_reporter"
const outputSinksKey contextKey = "codex_output_sinks"
const idleTimeoutKey contextKey = "codex_idle_timeout"

type PIDReporter func(int)

type OutputSinks struct {
	Stdout io.Writer
	Stderr io.Writer
}

func WithOutputPath(ctx context.Context, path string) context.Context {
	if strings.TrimSpace(path) == "" {
		return ctx
	}
	return context.WithValue(ctx, outputPathKey, path)
}

func WithPIDReporter(ctx context.Context, reporter PIDReporter) context.Context {
	if reporter == nil {
		return ctx
	}
	return context.WithValue(ctx, pidReporterKey, reporter)
}

func WithOutputSinks(ctx context.Context, stdout, stderr io.Writer) context.Context {
	if stdout == nil && stderr == nil {
		return ctx
	}
	return context.WithValue(ctx, outputSinksKey, OutputSinks{Stdout: stdout, Stderr: stderr})
}

func WithIdleTimeout(ctx context.Context, timeout time.Duration) context.Context {
	if timeout <= 0 {
		return ctx
	}
	return context.WithValue(ctx, idleTimeoutKey, timeout)
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

func pidReporterFromContext(ctx context.Context) PIDReporter {
	if ctx == nil {
		return nil
	}
	if value, ok := ctx.Value(pidReporterKey).(PIDReporter); ok {
		return value
	}
	return nil
}

func outputSinksFromContext(ctx context.Context) OutputSinks {
	if ctx == nil {
		return OutputSinks{}
	}
	if value, ok := ctx.Value(outputSinksKey).(OutputSinks); ok {
		return value
	}
	return OutputSinks{}
}

func idleTimeoutFromContext(ctx context.Context) time.Duration {
	if ctx == nil {
		return 0
	}
	if value, ok := ctx.Value(idleTimeoutKey).(time.Duration); ok {
		return value
	}
	return 0
}
