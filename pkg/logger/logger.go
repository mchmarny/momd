package logger

import (
	"log"
	"log/slog"
	"os"
	"strings"
)

const (
	// EnvVarLogLevel is the environment variable name for setting the log level.
	EnvVarLogLevel = "LOG_LEVEL"
)

// New creates a new structured logger instance and sets it as the default slog logger.
// The logger is configured based on the log level specified in the environment variable LOG_LEVEL.
// If the environment variable is not set or contains an unrecognized value, the default log level is Info.
func New(name, version string) *log.Logger {
	levelType := ParseLogLevel(os.Getenv(EnvVarLogLevel))
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     levelType,
		AddSource: true,
	})

	// Add module and version as attributes
	handlerWithAttrs := handler.WithAttrs([]slog.Attr{
		slog.String("name", name),
		slog.String("version", version),
	})

	// Set as the default slog logger so all slog.Info/Error calls use this handler
	slog.SetDefault(slog.New(handlerWithAttrs))

	return slog.NewLogLogger(handlerWithAttrs, levelType)
}

// ParseLogLevel converts a string representation of a log level into a slog.Level.
// Parameters:
//   - level: The log level as a string (e.g., "debug", "info", "warn", "error").
//
// Returns:
//   - slog.Level corresponding to the input string. Defaults to slog.LevelInfo for unrecognized strings.
func ParseLogLevel(level string) slog.Level {
	var lev slog.Level

	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		lev = slog.LevelDebug
	case "warn", "warning":
		lev = slog.LevelWarn
	case "error":
		lev = slog.LevelError
	default:
		lev = slog.LevelInfo
	}

	return lev
}
