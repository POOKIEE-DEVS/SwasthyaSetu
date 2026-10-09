// Package logging sets up structured logs: readable lines in development,
// one JSON object per line in production (LOG_JSON=true) for a log
// aggregator.
package logging

import (
	"io"
	"log/slog"
	"strings"
)

// New returns a logger writing to w at the given level ("DEBUG", "INFO",
// "WARNING" or "ERROR"; unknown levels mean INFO).
func New(w io.Writer, level string, json bool) *slog.Logger {
	opts := &slog.HandlerOptions{Level: ParseLevel(level)}
	if json {
		// "timestamp" and "message", the usual names for log aggregators.
		opts.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
			if len(groups) == 0 {
				switch a.Key {
				case slog.TimeKey:
					a.Key = "timestamp"
				case slog.MessageKey:
					a.Key = "message"
				}
			}
			return a
		}
		return slog.New(slog.NewJSONHandler(w, opts))
	}
	return slog.New(slog.NewTextHandler(w, opts))
}

// ParseLevel maps a LOG_LEVEL value to a slog level.
func ParseLevel(level string) slog.Level {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR", "CRITICAL":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Discard is a logger that drops everything, for tests.
func Discard() *slog.Logger { return slog.New(slog.DiscardHandler) }
