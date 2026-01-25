package log

import (
	"bytes"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestVerbosityToLevel(t *testing.T) {
	tests := []struct {
		verbosity int
		expected  zapcore.Level
	}{
		{0, zapcore.ErrorLevel},
		{-1, zapcore.ErrorLevel},
		{1, zapcore.WarnLevel},
		{2, zapcore.InfoLevel},
		{3, zapcore.DebugLevel},
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
		level    zapcore.Level
		expected int
	}{
		{zapcore.ErrorLevel, VerbosityError},
		{zapcore.WarnLevel, VerbosityWarn},
		{zapcore.InfoLevel, VerbosityInfo},
		{zapcore.DebugLevel, VerbosityDebug},
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
		level    zapcore.Level
		expected string
	}{
		{LevelTrace, "TRACE"},
		{zapcore.DebugLevel, "DEBUG"},
		{zapcore.InfoLevel, "INFO"},
		{zapcore.WarnLevel, "WARN"},
		{zapcore.ErrorLevel, "ERROR"},
	}

	for _, tt := range tests {
		got := LevelName(tt.level)
		if got != tt.expected {
			t.Errorf("LevelName(%v) = %q, want %q", tt.level, got, tt.expected)
		}
	}
}

// testLogger creates a test logger that writes to a buffer
func testLogger(buf *bytes.Buffer, level zapcore.Level) *zap.Logger {
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoder := zapcore.NewConsoleEncoder(encoderConfig)

	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(buf),
		level,
	)

	return zap.New(core)
}

func TestInit(t *testing.T) {
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
	testLog := testLogger(&buf, zapcore.InfoLevel)
	logger.Store(testLog)
	sugar.Store(testLog.Sugar())
	verbosity.Store(2)

	// V(2) should log at info level (v=2)
	V(2).Infow("should appear", "key", "value")
	if !strings.Contains(buf.String(), "should appear") {
		t.Errorf("V(2) should log when verbosity is 2, got: %s", buf.String())
	}

	buf.Reset()

	// V(3) should not log when verbosity is 2
	V(3).Infow("should not appear", "key", "value")
	if strings.Contains(buf.String(), "should not appear") {
		t.Errorf("V(3) should not log when verbosity is 2, got: %s", buf.String())
	}
}

func TestWith(t *testing.T) {
	var buf bytes.Buffer

	testLog := testLogger(&buf, zapcore.InfoLevel)
	logger.Store(testLog)
	sugar.Store(testLog.Sugar())

	componentLogger := With("component", "test")
	componentLogger.Infow("test message")

	if !strings.Contains(buf.String(), "component") || !strings.Contains(buf.String(), "test") {
		t.Errorf("With should add context, got: %s", buf.String())
	}
}

func TestComponent(t *testing.T) {
	var buf bytes.Buffer

	testLog := testLogger(&buf, zapcore.InfoLevel)
	logger.Store(testLog)
	sugar.Store(testLog.Sugar())

	componentLogger := Component("mycomponent")
	componentLogger.Infow("test message")

	if !strings.Contains(buf.String(), "component") || !strings.Contains(buf.String(), "mycomponent") {
		t.Errorf("Component should add component context, got: %s", buf.String())
	}
}

func TestJSON(t *testing.T) {
	var buf bytes.Buffer

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoder := zapcore.NewJSONEncoder(encoderConfig)

	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(&buf),
		zapcore.InfoLevel,
	)

	l := zap.New(core)
	l.Sugar().Infow("test", "key", "value")

	// JSON output should contain the structured fields
	if !strings.Contains(buf.String(), `"key":"value"`) {
		t.Errorf("JSON handler should output JSON, got: %s", buf.String())
	}
}
