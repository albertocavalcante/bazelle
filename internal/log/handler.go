package log

import (
	"context"
	"io"
	"log/slog"
	"os"
)

// HandlerOptions configures the log handler.
type HandlerOptions struct {
	Level     slog.Leveler
	Format    string // "text" or "json"
	Output    io.Writer
	AddSource bool
}

// NewHandler creates appropriate handler based on options.
func NewHandler(opts HandlerOptions) slog.Handler {
	if opts.Output == nil {
		opts.Output = os.Stderr // Always stderr, never stdout
	}

	handlerOpts := &slog.HandlerOptions{
		Level:       opts.Level,
		AddSource:   opts.AddSource,
		ReplaceAttr: replaceLevelNames,
	}

	if opts.Format == "json" {
		return slog.NewJSONHandler(opts.Output, handlerOpts)
	}
	return slog.NewTextHandler(opts.Output, handlerOpts)
}

// replaceLevelNames customizes level display (TRACE, etc.).
func replaceLevelNames(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.LevelKey {
		level, ok := a.Value.Any().(slog.Level)
		if ok {
			a.Value = slog.StringValue(LevelName(level))
		}
	}
	return a
}

// discardHandler is a handler that discards all log records.
type discardHandler struct{}

func (discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (d discardHandler) WithAttrs([]slog.Attr) slog.Handler      { return d }
func (d discardHandler) WithGroup(string) slog.Handler           { return d }
