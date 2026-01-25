package log

import (
	"context"
	"log/slog"
	"os"
	"sync/atomic"
)

var (
	logger    atomic.Pointer[slog.Logger]
	level     *slog.LevelVar
	verbosity atomic.Int32
)

func init() {
	// Initialize with default logger (warnings only) before Init is called
	level = new(slog.LevelVar)
	level.Set(slog.LevelWarn)
	defaultLogger := slog.New(NewHandler(HandlerOptions{
		Level:  level,
		Format: "text",
		Output: os.Stderr,
	}))
	logger.Store(defaultLogger)
}

// Init initializes the global logger (call once at startup).
func Init(v int, format string) {
	verbosity.Store(int32(v))
	level.Set(VerbosityToLevel(v))

	handler := NewHandler(HandlerOptions{
		Level:  level,
		Format: format,
		Output: os.Stderr,
	})
	newLogger := slog.New(handler)
	logger.Store(newLogger)
	slog.SetDefault(newLogger)
}

// SetVerbosity changes verbosity at runtime.
func SetVerbosity(v int) {
	verbosity.Store(int32(v))
	level.Set(VerbosityToLevel(v))
}

// Verbosity returns the current verbosity level.
func Verbosity() int {
	return int(verbosity.Load())
}

// Logger returns the current logger instance.
func Logger() *slog.Logger {
	return logger.Load()
}

// Error logs at error level (v=0).
func Error(msg string, args ...any) {
	logger.Load().Error(msg, args...)
}

// Warn logs at warn level (v=1).
func Warn(msg string, args ...any) {
	logger.Load().Warn(msg, args...)
}

// Info logs at info level (v=2).
func Info(msg string, args ...any) {
	logger.Load().Info(msg, args...)
}

// Debug logs at debug level (v=3).
func Debug(msg string, args ...any) {
	logger.Load().Debug(msg, args...)
}

// Trace logs at trace level (v=4).
func Trace(msg string, args ...any) {
	logger.Load().Log(context.Background(), LevelTrace, msg, args...)
}

// V returns a logger that only logs if verbosity >= level.
// Usage: log.V(3).Info("detailed", "key", value)
func V(v int) *slog.Logger {
	if int(verbosity.Load()) >= v {
		return logger.Load()
	}
	return slog.New(discardHandler{})
}

// With returns a logger with additional context.
func With(args ...any) *slog.Logger {
	return logger.Load().With(args...)
}

// Component returns a logger tagged with component name.
func Component(name string) *slog.Logger {
	return logger.Load().With("component", name)
}
