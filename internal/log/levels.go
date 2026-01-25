// Package log provides structured logging with verbosity levels for Bazelle.
// It wraps uber-go/zap and follows kubectl/klog patterns.
package log

import "go.uber.org/zap/zapcore"

// LevelTrace is a custom trace level (more verbose than debug).
// Zap doesn't have trace, so we use a custom level below Debug (-1).
const LevelTrace = zapcore.Level(-2)

// Verbosity level constants for documentation and reference.
const (
	VerbosityError = 0 // Errors only (quiet)
	VerbosityWarn  = 1 // + Warnings
	VerbosityInfo  = 2 // + Info (config loaded, languages enabled, summaries)
	VerbosityDebug = 3 // + Debug (files scanned, parse details, timing)
	VerbosityTrace = 4 // + Trace (function entry/exit, full data dumps)
)

// VerbosityToLevel maps -v=N to zap level.
func VerbosityToLevel(v int) zapcore.Level {
	switch {
	case v <= 0:
		return zapcore.ErrorLevel
	case v == 1:
		return zapcore.WarnLevel
	case v == 2:
		return zapcore.InfoLevel
	case v == 3:
		return zapcore.DebugLevel
	default:
		return LevelTrace
	}
}

// LevelToVerbosity maps zap level to -v=N (for display).
func LevelToVerbosity(l zapcore.Level) int {
	switch {
	case l >= zapcore.ErrorLevel:
		return VerbosityError
	case l >= zapcore.WarnLevel:
		return VerbosityWarn
	case l >= zapcore.InfoLevel:
		return VerbosityInfo
	case l >= zapcore.DebugLevel:
		return VerbosityDebug
	default:
		return VerbosityTrace
	}
}

// LevelName returns the name for a zap level, including custom levels.
func LevelName(l zapcore.Level) string {
	if l == LevelTrace {
		return "TRACE"
	}
	return l.CapitalString()
}
