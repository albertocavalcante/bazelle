// Package log provides structured logging with verbosity levels for Bazelle.
// It wraps Go's log/slog package and follows kubectl/klog patterns.
package log

import "log/slog"

// LevelTrace is a custom trace level (more verbose than debug).
const LevelTrace = slog.Level(-8)

// Verbosity level constants for documentation and reference.
const (
	VerbosityError = 0 // Errors only (quiet)
	VerbosityWarn  = 1 // + Warnings
	VerbosityInfo  = 2 // + Info (config loaded, languages enabled, summaries)
	VerbosityDebug = 3 // + Debug (files scanned, parse details, timing)
	VerbosityTrace = 4 // + Trace (function entry/exit, full data dumps)
)

// VerbosityToLevel maps -v=N to slog level.
func VerbosityToLevel(v int) slog.Level {
	switch {
	case v <= 0:
		return slog.LevelError
	case v == 1:
		return slog.LevelWarn
	case v == 2:
		return slog.LevelInfo
	case v == 3:
		return slog.LevelDebug
	default:
		return LevelTrace
	}
}

// LevelToVerbosity maps slog level to -v=N (for display).
func LevelToVerbosity(l slog.Level) int {
	switch {
	case l >= slog.LevelError:
		return VerbosityError
	case l >= slog.LevelWarn:
		return VerbosityWarn
	case l >= slog.LevelInfo:
		return VerbosityInfo
	case l >= slog.LevelDebug:
		return VerbosityDebug
	default:
		return VerbosityTrace
	}
}

// LevelName returns the name for a slog level, including custom levels.
func LevelName(l slog.Level) string {
	switch l {
	case LevelTrace:
		return "TRACE"
	case slog.LevelDebug:
		return "DEBUG"
	case slog.LevelInfo:
		return "INFO"
	case slog.LevelWarn:
		return "WARN"
	case slog.LevelError:
		return "ERROR"
	default:
		return l.String()
	}
}
