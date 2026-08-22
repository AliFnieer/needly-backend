package observability

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// Logger wraps slog with a consistent interface for the application.
type Logger struct {
	*slog.Logger
}

// NewLogger creates a structured logger with the given level and output.
// level can be "debug", "info", "warn", or "error".
func NewLogger(level string, output io.Writer) *Logger {
	if output == nil {
		output = os.Stdout
	}

	var slogLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		slogLevel = slog.LevelDebug
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(output, &slog.HandlerOptions{
		Level: slogLevel,
	})

	return &Logger{
		Logger: slog.New(handler),
	}
}

// With returns a logger with additional key-value attributes.
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		Logger: l.Logger.With(args...),
	}
}
