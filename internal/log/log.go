package log

import (
	"os"
	"sync/atomic"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger    atomic.Pointer[zap.Logger]
	sugar     atomic.Pointer[zap.SugaredLogger]
	atomLevel zap.AtomicLevel
	verbosity atomic.Int32
)

func init() {
	// Default logger before Init() is called
	atomLevel = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	defaultLogger := newLogger("text")
	logger.Store(defaultLogger)
	sugar.Store(defaultLogger.Sugar())
}

// Init initializes the global logger (call once at startup).
func Init(v int, format string) {
	verbosity.Store(int32(v))
	atomLevel.SetLevel(VerbosityToLevel(v))

	newLog := newLogger(format)
	logger.Store(newLog)
	sugar.Store(newLog.Sugar())
}

func newLogger(format string) *zap.Logger {
	var encoder zapcore.Encoder

	if format == "json" {
		encoderConfig := zap.NewProductionEncoderConfig()
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		encoderConfig.EncodeLevel = customLevelEncoder
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		// Development encoder with colors
		devConfig := zap.NewDevelopmentEncoderConfig()
		devConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		devConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		encoder = zapcore.NewConsoleEncoder(devConfig)
	}

	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stderr),
		atomLevel,
	)

	return zap.New(core)
}

func customLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(LevelName(l))
}

// SetVerbosity changes verbosity at runtime.
func SetVerbosity(v int) {
	verbosity.Store(int32(v))
	atomLevel.SetLevel(VerbosityToLevel(v))
}

// Verbosity returns the current verbosity level.
func Verbosity() int {
	return int(verbosity.Load())
}

// Logger returns the underlying zap logger.
func Logger() *zap.Logger {
	return logger.Load()
}

// Error logs at error level (v=0).
func Error(msg string, args ...any) {
	sugar.Load().Errorw(msg, args...)
}

// Warn logs at warn level (v=1).
func Warn(msg string, args ...any) {
	sugar.Load().Warnw(msg, args...)
}

// Info logs at info level (v=2).
func Info(msg string, args ...any) {
	sugar.Load().Infow(msg, args...)
}

// Debug logs at debug level (v=3).
func Debug(msg string, args ...any) {
	sugar.Load().Debugw(msg, args...)
}

// Trace logs at trace level (v=4).
func Trace(msg string, args ...any) {
	if verbosity.Load() >= VerbosityTrace {
		sugar.Load().Debugw("[TRACE] "+msg, args...)
	}
}

// V returns a logger that only logs if verbosity >= level.
// Usage: log.V(3).Infow("detailed", "key", value)
func V(v int) *zap.SugaredLogger {
	if int(verbosity.Load()) >= v {
		return sugar.Load()
	}
	return zap.NewNop().Sugar()
}

// With returns a logger with additional context.
func With(args ...any) *zap.SugaredLogger {
	return sugar.Load().With(args...)
}

// Component returns a logger tagged with component name.
func Component(name string) *zap.SugaredLogger {
	return sugar.Load().With("component", name)
}
