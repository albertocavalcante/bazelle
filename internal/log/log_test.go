package log

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestVerbosityToLevel(t *testing.T) {
	tests := []struct {
		verbosity int
		expected  slog.Level
	}{
		{0, slog.LevelError},
		{-1, slog.LevelError},
		{1, slog.LevelWarn},
		{2, slog.LevelInfo},
		{3, slog.LevelDebug},
		{4, LevelTrace},
		{5, LevelTrace}, // anything > 4 maps to trace
	}

	for _, tt := range tests {
		got := VerbosityToLevel(tt.verbosity)
		if got != tt.expected {
			t.Errorf("VerbosityToLevel(%d) = %v, want %v", tt.verbosity, got, tt.expected)
		}
	}
}

func TestLevelToVerbosity(t *testing.T) {
	tests := []struct {
		level    slog.Level
		expected int
	}{
		{slog.LevelError, VerbosityError},
		{slog.LevelWarn, VerbosityWarn},
		{slog.LevelInfo, VerbosityInfo},
		{slog.LevelDebug, VerbosityDebug},
		{LevelTrace, VerbosityTrace},
	}

	for _, tt := range tests {
		got := LevelToVerbosity(tt.level)
		if got != tt.expected {
			t.Errorf("LevelToVerbosity(%v) = %d, want %d", tt.level, got, tt.expected)
		}
	}
}

func TestLevelName(t *testing.T) {
	tests := []struct {
		level    slog.Level
		expected string
	}{
		{LevelTrace, "TRACE"},
		{slog.LevelDebug, "DEBUG"},
		{slog.LevelInfo, "INFO"},
		{slog.LevelWarn, "WARN"},
		{slog.LevelError, "ERROR"},
	}

	for _, tt := range tests {
		got := LevelName(tt.level)
		if got != tt.expected {
			t.Errorf("LevelName(%v) = %q, want %q", tt.level, got, tt.expected)
		}
	}
}

func TestInit(t *testing.T) {
	var buf bytes.Buffer

	// Initialize with a custom output for testing
	level = new(slog.LevelVar)
	handler := NewHandler(HandlerOptions{
		Level:  level,
		Format: "text",
		Output: &buf,
	})
	newLogger := slog.New(handler)
	logger.Store(newLogger)

	// Test that Init sets verbosity correctly
	Init(2, "text")
	if Verbosity() != 2 {
		t.Errorf("Verbosity() = %d, want 2", Verbosity())
	}
}

func TestSetVerbosity(t *testing.T) {
	Init(1, "text")

	SetVerbosity(3)
	if Verbosity() != 3 {
		t.Errorf("Verbosity() = %d, want 3", Verbosity())
	}

	SetVerbosity(0)
	if Verbosity() != 0 {
		t.Errorf("Verbosity() = %d, want 0", Verbosity())
	}
}

func TestV(t *testing.T) {
	var buf bytes.Buffer

	// Setup logger with custom output
	level = new(slog.LevelVar)
	level.Set(VerbosityToLevel(2))
	handler := NewHandler(HandlerOptions{
		Level:  level,
		Format: "text",
		Output: &buf,
	})
	newLogger := slog.New(handler)
	logger.Store(newLogger)
	verbosity.Store(2)

	// V(2) should log at info level (v=2)
	V(2).Info("should appear", "key", "value")
	if !strings.Contains(buf.String(), "should appear") {
		t.Errorf("V(2) should log when verbosity is 2, got: %s", buf.String())
	}

	buf.Reset()

	// V(3) should not log when verbosity is 2
	V(3).Info("should not appear", "key", "value")
	if strings.Contains(buf.String(), "should not appear") {
		t.Errorf("V(3) should not log when verbosity is 2, got: %s", buf.String())
	}
}

func TestWith(t *testing.T) {
	var buf bytes.Buffer

	level = new(slog.LevelVar)
	level.Set(slog.LevelInfo)
	handler := NewHandler(HandlerOptions{
		Level:  level,
		Format: "text",
		Output: &buf,
	})
	newLogger := slog.New(handler)
	logger.Store(newLogger)

	componentLogger := With("component", "test")
	componentLogger.Info("test message")

	if !strings.Contains(buf.String(), "component=test") {
		t.Errorf("With should add context, got: %s", buf.String())
	}
}

func TestComponent(t *testing.T) {
	var buf bytes.Buffer

	level = new(slog.LevelVar)
	level.Set(slog.LevelInfo)
	handler := NewHandler(HandlerOptions{
		Level:  level,
		Format: "text",
		Output: &buf,
	})
	newLogger := slog.New(handler)
	logger.Store(newLogger)

	componentLogger := Component("mycomponent")
	componentLogger.Info("test message")

	if !strings.Contains(buf.String(), "component=mycomponent") {
		t.Errorf("Component should add component context, got: %s", buf.String())
	}
}

func TestNewHandler_JSON(t *testing.T) {
	var buf bytes.Buffer

	handler := NewHandler(HandlerOptions{
		Level:  slog.LevelInfo,
		Format: "json",
		Output: &buf,
	})

	l := slog.New(handler)
	l.Info("test", "key", "value")

	// JSON output should contain the structured fields
	if !strings.Contains(buf.String(), `"key":"value"`) {
		t.Errorf("JSON handler should output JSON, got: %s", buf.String())
	}
}

func TestNewHandler_DefaultOutput(t *testing.T) {
	// Ensure we can create a handler with nil output (defaults to stderr)
	handler := NewHandler(HandlerOptions{
		Level:  slog.LevelInfo,
		Format: "text",
		Output: nil, // should default to stderr
	})

	if handler == nil {
		t.Error("NewHandler should not return nil")
	}
}
