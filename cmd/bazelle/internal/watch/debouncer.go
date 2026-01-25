// Package watch implements file watching for automatic BUILD file updates.
package watch

import (
	"sync"
	"time"
)

// MaxPendingDirs is the maximum number of directories that can be pending.
// If this limit is reached, a flush is triggered immediately to prevent
// unbounded memory growth from rapid file creation.
const MaxPendingDirs = 1000

// Debouncer coalesces rapid file change events into batched directory updates.
// It groups events within a time window to avoid triggering multiple updates
// when files are saved rapidly (e.g., IDE autosave, formatter runs).
type Debouncer struct {
	mu      sync.Mutex
	pending map[string]struct{} // set of pending directories
	timer   *time.Timer
	window  time.Duration
	onFlush func(dirs []string)
	stopped bool
}

// NewDebouncer creates a debouncer with the given window duration.
// The onFlush callback is called with the list of affected directories
// after the window expires with no new events.
func NewDebouncer(window time.Duration, onFlush func(dirs []string)) *Debouncer {
	return &Debouncer{
		pending: make(map[string]struct{}),
		window:  window,
		onFlush: onFlush,
	}
}

// Add records a change in the given directory.
// Multiple calls with the same directory within the window are coalesced.
func (d *Debouncer) Add(dir string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.stopped {
		return
	}

	// Record directory if not already pending
	d.pending[dir] = struct{}{}

	// Check if we've hit the pending limit - force immediate flush
	if len(d.pending) >= MaxPendingDirs {
		if d.timer != nil {
			d.timer.Stop()
			d.timer = nil
		}
		d.flushLocked()
		return
	}

	// Reset or start timer.
	// Note: timer.Stop() may return false if the timer has already fired,
	// meaning flush() may already be queued to run. This is safe because
	// flush() checks len(d.pending) and exits early if empty.
	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(d.window, d.flush)
}

// flush is called when the timer expires.
func (d *Debouncer) flush() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.flushLocked()
}

// flushLocked performs the flush while holding the lock.
// Caller must hold d.mu.
func (d *Debouncer) flushLocked() {
	if d.stopped || len(d.pending) == 0 {
		return
	}

	// Collect directories
	dirs := make([]string, 0, len(d.pending))
	for dir := range d.pending {
		dirs = append(dirs, dir)
	}

	// Clear pending
	d.pending = make(map[string]struct{})

	// Release lock before calling handler to prevent deadlocks
	d.mu.Unlock()

	// Call handler outside lock
	if d.onFlush != nil {
		d.onFlush(dirs)
	}

	// Re-acquire lock (caller expects it held via defer)
	d.mu.Lock()
}

// FlushNow immediately flushes any pending directories without waiting
// for the timer. This is useful for graceful shutdown.
func (d *Debouncer) FlushNow() {
	d.mu.Lock()
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}

	// Collect and clear while holding lock to prevent races
	if d.stopped || len(d.pending) == 0 {
		d.mu.Unlock()
		return
	}

	dirs := make([]string, 0, len(d.pending))
	for dir := range d.pending {
		dirs = append(dirs, dir)
	}
	d.pending = make(map[string]struct{})
	d.mu.Unlock()

	// Call handler outside lock
	if d.onFlush != nil {
		d.onFlush(dirs)
	}
}

// Stop stops the debouncer. Any pending directories are flushed.
func (d *Debouncer) Stop() {
	d.mu.Lock()
	d.stopped = true
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}

	// Collect directories while holding lock
	dirs := make([]string, 0, len(d.pending))
	for dir := range d.pending {
		dirs = append(dirs, dir)
	}
	d.pending = make(map[string]struct{})
	d.mu.Unlock()

	// Call handler outside lock
	if len(dirs) > 0 && d.onFlush != nil {
		d.onFlush(dirs)
	}
}

// PendingCount returns the number of directories waiting to be flushed.
func (d *Debouncer) PendingCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.pending)
}
