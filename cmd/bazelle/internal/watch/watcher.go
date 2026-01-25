package watch

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/albertocavalcante/bazelle/cmd/bazelle/internal/incremental"
	"github.com/albertocavalcante/bazelle/cmd/bazelle/internal/langs"
	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/bazelbuild/bazel-gazelle/runner"
	"github.com/fsnotify/fsnotify"
)

// Config configures the watcher.
type Config struct {
	Root            string
	Languages       []language.Language
	LangFilter      []string // filter by language name (nil = all)
	Debounce        int      // debounce window in milliseconds
	Verbose         bool
	NoColor         bool
	JSON            bool
	GazelleDefaults []string
}

// Watcher watches for file changes and updates BUILD files.
type Watcher struct {
	config     Config
	fsWatcher  *fsnotify.Watcher
	tracker    *incremental.Tracker
	debouncer  *Debouncer
	logger     *Logger
	extensions map[string]bool
	ignoreDirs map[string]bool

	// gazelleMu prevents concurrent Gazelle runs
	gazelleMu sync.Mutex
}

// New creates a new watcher with the given configuration.
func New(cfg Config) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	// Build extension filter from shared config
	extensions := langs.ExtensionSet(cfg.LangFilter)

	// Build ignore directory filter from shared config
	ignoreDirs := langs.IgnoreDirSet(nil)

	logger := NewLogger(LoggerConfig{
		Verbose: cfg.Verbose,
		NoColor: cfg.NoColor,
		JSON:    cfg.JSON,
	})

	tracker := incremental.NewTracker(cfg.Root, cfg.LangFilter)

	w := &Watcher{
		config:     cfg,
		fsWatcher:  fsWatcher,
		tracker:    tracker,
		logger:     logger,
		extensions: extensions,
		ignoreDirs: ignoreDirs,
	}

	return w, nil
}

// Run starts the watch loop. It blocks until the context is cancelled.
func (w *Watcher) Run(ctx context.Context) error {
	// Setup debouncer
	debounceWindow := time.Duration(w.config.Debounce) * time.Millisecond
	if debounceWindow <= 0 {
		debounceWindow = 500 * time.Millisecond
	}
	w.debouncer = NewDebouncer(debounceWindow, w.handleChangedDirs)
	defer w.debouncer.Stop()

	// Add workspace root recursively
	if err := w.addRecursive(w.config.Root); err != nil {
		return fmt.Errorf("failed to watch workspace: %w", err)
	}

	// Log ready message
	// Note: TrackedFileCount may be 0 on first run before any state exists
	fileCount := w.tracker.TrackedFileCount()
	w.logger.Ready(fileCount, w.config.LangFilter, w.config.Root)

	// Main event loop
	for {
		select {
		case <-ctx.Done():
			w.logger.Shutdown()
			return nil

		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return nil
			}
			w.handleEvent(event)

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return nil
			}
			w.logger.Error(err)
		}
	}
}

// addRecursive adds a directory and all subdirectories to the watcher.
func (w *Watcher) addRecursive(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Log permission errors in verbose mode, skip silently otherwise
			if os.IsPermission(err) {
				if w.config.Verbose {
					w.logger.Error(fmt.Errorf("permission denied: %s", path))
				}
				return nil
			}
			// Log other errors but continue
			w.logger.Error(fmt.Errorf("walk error at %s: %w", path, err))
			return nil
		}

		if !d.IsDir() {
			return nil
		}

		// Check if directory should be ignored
		name := d.Name()
		for prefix := range w.ignoreDirs {
			if strings.HasPrefix(name, prefix) {
				return filepath.SkipDir
			}
		}

		// Add directory to watcher
		if err := w.fsWatcher.Add(path); err != nil {
			// Check for inotify limit errors
			if isWatchLimitError(err) {
				return fmt.Errorf("inotify watch limit reached for %s: %w\n"+
					"Increase limit with: sudo sysctl fs.inotify.max_user_watches=524288", path, err)
			}
			// Log other errors in verbose mode but continue
			if w.config.Verbose {
				w.logger.Error(fmt.Errorf("failed to watch %s: %w", path, err))
			}
			return nil
		}

		return nil
	})
}

// isWatchLimitError checks if an error is due to inotify watch limits.
func isWatchLimitError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "no space left on device") ||
		strings.Contains(errStr, "too many open files")
}

// handleEvent processes a single filesystem event.
func (w *Watcher) handleEvent(event fsnotify.Event) {
	path := event.Name

	// Handle directory events (create, rename)
	if event.Has(fsnotify.Create) {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			// Check if should be ignored
			name := filepath.Base(path)
			for prefix := range w.ignoreDirs {
				if strings.HasPrefix(name, prefix) {
					return
				}
			}
			// Add new directory to watcher
			if err := w.addRecursive(path); err != nil {
				w.logger.Error(fmt.Errorf("failed to watch new directory %s: %w", path, err))
			}
			return
		}
	}

	// Handle directory rename - need to re-add watches
	if event.Has(fsnotify.Rename) {
		// The old path is gone, fsnotify will auto-remove it
		// If a new directory appeared at a different location, it will get a Create event
		// So we just need to handle file renames here
	}

	// Filter by extension
	ext := filepath.Ext(path)
	if !w.extensions[ext] {
		return
	}

	// Determine change type and log
	var changeType ChangeType
	switch {
	case event.Has(fsnotify.Create):
		changeType = ChangeAdded
	case event.Has(fsnotify.Write):
		changeType = ChangeModified
	case event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename):
		changeType = ChangeDeleted
	default:
		return // Ignore chmod events
	}

	w.logger.FileChanged(path, changeType)

	// Get relative directory and debounce
	relPath, err := filepath.Rel(w.config.Root, path)
	if err != nil {
		return
	}
	dir := filepath.Dir(relPath)

	w.debouncer.Add(dir)
}

// handleChangedDirs is called when the debouncer flushes.
// It runs gazelle on the affected directories.
func (w *Watcher) handleChangedDirs(dirs []string) {
	if len(dirs) == 0 {
		return
	}

	// Prevent concurrent Gazelle runs
	w.gazelleMu.Lock()
	defer w.gazelleMu.Unlock()

	// Sort directories for consistent output
	slices.Sort(dirs)

	w.logger.Updating(dirs)

	// Convert to targets
	targets := make([]string, len(dirs))
	for i, dir := range dirs {
		if dir == "." {
			targets[i] = "//:all"
		} else {
			targets[i] = "//" + dir + ":all"
		}
	}

	// Build gazelle arguments
	args := []string{"update"}
	args = append(args, w.config.GazelleDefaults...)
	args = append(args, targets...)

	// Run gazelle
	if err := runner.Run(w.config.Languages, w.config.Root, args...); err != nil {
		w.logger.Error(fmt.Errorf("gazelle failed: %w", err))
		return
	}

	// Refresh tracker state
	ctx := context.Background()
	if err := w.tracker.Refresh(ctx); err != nil {
		w.logger.Error(fmt.Errorf("failed to update state: %w", err))
		return
	}

	// Log success for each directory, checking for actual BUILD file
	for _, dir := range dirs {
		buildFile := w.findBuildFile(dir)
		w.logger.Updated(buildFile)
	}
}

// findBuildFile returns the path to the BUILD file in a directory,
// checking for both BUILD.bazel and BUILD.
func (w *Watcher) findBuildFile(dir string) string {
	// Check BUILD.bazel first (preferred)
	bazelPath := filepath.Join(w.config.Root, dir, "BUILD.bazel")
	if _, err := os.Stat(bazelPath); err == nil {
		return filepath.Join(dir, "BUILD.bazel")
	}

	// Check BUILD
	buildPath := filepath.Join(w.config.Root, dir, "BUILD")
	if _, err := os.Stat(buildPath); err == nil {
		return filepath.Join(dir, "BUILD")
	}

	// Default to BUILD.bazel if neither exists (new file will be created)
	return filepath.Join(dir, "BUILD.bazel")
}

// Close closes the watcher and releases resources.
func (w *Watcher) Close() error {
	if w.fsWatcher != nil {
		return w.fsWatcher.Close()
	}
	return nil
}

// ErrWatchLimitReached is returned when the OS watch limit is exceeded.
var ErrWatchLimitReached = errors.New("filesystem watch limit reached")
