package logging

import (
	"log/slog"
	"os"
	"strings"
	"time"
)

func NewLogger(level, format string) *slog.Logger {
	lvl := slog.LevelInfo
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	}

	handlerOptions := &slog.HandlerOptions{
		Level:     lvl,
		AddSource: false,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{Key: "time", Value: slog.StringValue(a.Value.Time().UTC().Format(time.RFC3339Nano))}
			}
			return a
		},
	}

	switch strings.ToLower(strings.TrimSpace(format)) {
	case "text":
		return slog.New(slog.NewTextHandler(os.Stderr, handlerOptions))
	default:
		return slog.New(slog.NewJSONHandler(os.Stderr, handlerOptions))
	}
}
