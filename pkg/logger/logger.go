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

// NewStructuredLogger creates a new structured logger with the specified log level
// Defined module name and version are included in the logger's context.
// AddSource is enabled for debug level logging only.
// Parameters:
//   - module: The name of the module/application using the logger.
//   - version: The version of the module/application (e.g., "v1.0.0").
//   - level: The log level as a string (e.g., "debug", "info", "warn", "error").
//
// Returns:
//   - *slog.Logger: A pointer to the configured slog.Logger instance.
func NewStructuredLogger(module, version, level string) *slog.Logger {
	lev := ParseLogLevel(level)
	addSource := lev <= slog.LevelDebug

	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level:     lev,
		AddSource: addSource,
	})).With("module", module, "version", version)
}

// NewLogLogger creates a new standard library log.Logger that writes logs
// using the slog package with the specified log level.
// Parameters:
//   - level: The log level as a slog.Level.
//
// Returns:
//   - *log.Logger: A pointer to the configured log.Logger instance.
func NewLogLogger(level slog.Level, withSource bool) *log.Logger {
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level:     level,
		AddSource: withSource,
	})

	return slog.NewLogLogger(handler, level)
}

// SetDefaultLogger initializes the structured logger with the
// appropriate log level and sets it as the default logger.
// Defined module name and version are included in the logger's context.
// Parameters:
//   - module: The name of the module/application using the logger.
//   - version: The version of the module/application (e.g., "v1.0.0").
//
// Derives log level from the LOG_LEVEL environment variable.
func SetDefaultLogger(module, version string) {
	SetDefaultLoggerWithLevel(module, version, os.Getenv(EnvVarLogLevel))
}

// SetDefaultLoggerWithLevel initializes the structured logger with the specified log level
// Defined module name and version are included in the logger's context.
// Parameters:
//   - module: The name of the module/application using the logger.
//   - version: The version of the module/application (e.g., "v1.0.0").
//   - level: The log level as a string (e.g., "debug", "info", "warn", "error").
func SetDefaultLoggerWithLevel(module, version, level string) {
	slog.SetDefault(NewStructuredLogger(module, version, level))
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
