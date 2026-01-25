package watch

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"golang.org/x/term"
)

// ChangeType represents the type of file change.
type ChangeType string

const (
	ChangeAdded    ChangeType = "+"
	ChangeModified ChangeType = "~"
	ChangeDeleted  ChangeType = "-"
)

// Logger handles watch mode output formatting.
type Logger struct {
	writer  io.Writer
	isTTY   bool
	verbose bool
	noColor bool
	jsonOut bool

	statsMu sync.Mutex
	stats   WatchStats
}

// WatchStats tracks statistics for the watch session.
type WatchStats struct {
	UpdateCount int
	ErrorCount  int
	StartTime   time.Time
}

// LoggerConfig configures the logger.
type LoggerConfig struct {
	Writer  io.Writer
	Verbose bool
	NoColor bool
	JSON    bool
}

// NewLogger creates a new logger with the given configuration.
func NewLogger(cfg LoggerConfig) *Logger {
	writer := cfg.Writer
	if writer == nil {
		writer = os.Stdout
	}

	isTTY := false
	if f, ok := writer.(*os.File); ok {
		isTTY = term.IsTerminal(int(f.Fd()))
	}

	return &Logger{
		writer:  writer,
		isTTY:   isTTY,
		verbose: cfg.Verbose,
		noColor: cfg.NoColor,
		jsonOut: cfg.JSON,
		stats: WatchStats{
			StartTime: time.Now(),
		},
	}
}

// Ready logs the initial ready message.
func (l *Logger) Ready(fileCount int, languages []string, path string) {
	if l.jsonOut {
		l.writeJSON(map[string]any{
			"event":     "ready",
			"files":     fileCount,
			"languages": languages,
			"path":      path,
		})
		return
	}

	l.printf("bazelle: watching %d files in %s\n", fileCount, path)
	if len(languages) > 0 {
		l.printf("bazelle: languages: ")
		for i, lang := range languages {
			if i > 0 {
				l.printf(", ")
			}
			l.printf("%s", lang)
		}
		l.println()
	}
	l.println("bazelle: ready")
	l.println()
}

// FileChanged logs a file change event.
func (l *Logger) FileChanged(path string, change ChangeType) {
	if l.jsonOut {
		l.writeJSON(map[string]any{
			"event":  "file_changed",
			"path":   path,
			"change": string(change),
			"time":   time.Now().Format(time.RFC3339),
		})
		return
	}

	if l.verbose {
		l.printf("[%s] %s %s\n", l.timestamp(), l.colorize(string(change), change), path)
	}
}

// Updating logs that an update is starting.
func (l *Logger) Updating(dirs []string) {
	if l.jsonOut {
		l.writeJSON(map[string]any{
			"event": "updating",
			"dirs":  dirs,
			"time":  time.Now().Format(time.RFC3339),
		})
		return
	}

	if len(dirs) == 1 {
		l.printf("[%s] updating //%s:all...\n", l.timestamp(), dirs[0])
	} else {
		l.printf("[%s] updating %d directories...\n", l.timestamp(), len(dirs))
	}
}

// Updated logs a successful update.
func (l *Logger) Updated(buildFile string) {
	l.statsMu.Lock()
	l.stats.UpdateCount++
	l.statsMu.Unlock()

	if l.jsonOut {
		l.writeJSON(map[string]any{
			"event":      "updated",
			"build_file": buildFile,
			"time":       time.Now().Format(time.RFC3339),
		})
		return
	}

	checkmark := l.colorize("\u2713", ChangeAdded) // checkmark
	l.printf("[%s] %s %s updated\n", l.timestamp(), checkmark, buildFile)
}

// Error logs an error.
func (l *Logger) Error(err error) {
	l.statsMu.Lock()
	l.stats.ErrorCount++
	l.statsMu.Unlock()

	if l.jsonOut {
		l.writeJSON(map[string]any{
			"event": "error",
			"error": err.Error(),
			"time":  time.Now().Format(time.RFC3339),
		})
		return
	}

	xmark := l.colorize("\u2717", ChangeDeleted) // xmark
	l.printf("[%s] %s error: %v\n", l.timestamp(), xmark, err)
}

// Shutdown logs the shutdown message with statistics.
func (l *Logger) Shutdown() {
	l.statsMu.Lock()
	stats := l.stats
	l.statsMu.Unlock()

	if l.jsonOut {
		l.writeJSON(map[string]any{
			"event":    "shutdown",
			"updates":  stats.UpdateCount,
			"errors":   stats.ErrorCount,
			"duration": time.Since(stats.StartTime).String(),
		})
		return
	}

	l.println()
	l.printf("bazelle: shutting down (%d updates, %d errors)\n",
		stats.UpdateCount, stats.ErrorCount)
}

// Stats returns the current watch statistics.
func (l *Logger) Stats() WatchStats {
	l.statsMu.Lock()
	defer l.statsMu.Unlock()
	return l.stats
}

// timestamp returns the current time formatted as HH:MM:SS.
func (l *Logger) timestamp() string {
	return time.Now().Format("15:04:05")
}

// colorize applies ANSI color codes based on change type.
func (l *Logger) colorize(s string, change ChangeType) string {
	if l.noColor || !l.isTTY {
		return s
	}

	var color string
	switch change {
	case ChangeAdded:
		color = "\033[32m" // green
	case ChangeModified:
		color = "\033[33m" // yellow
	case ChangeDeleted:
		color = "\033[31m" // red
	default:
		return s
	}
	return color + s + "\033[0m"
}

// writeJSON writes a JSON object to the output.
func (l *Logger) writeJSON(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		// Write a minimal error event so tooling knows something went wrong
		l.println(`{"event":"internal_error","error":"json marshal failed"}`)
		return
	}
	l.println(string(data))
}

// printf writes a formatted string to the writer, ignoring errors.
// Logging output errors are intentionally ignored as they are informational.
func (l *Logger) printf(format string, args ...any) {
	_, _ = fmt.Fprintf(l.writer, format, args...)
}

// println writes a line to the writer, ignoring errors.
// Logging output errors are intentionally ignored as they are informational.
func (l *Logger) println(args ...any) {
	_, _ = fmt.Fprintln(l.writer, args...)
}
