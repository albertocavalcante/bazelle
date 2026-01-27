package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// DefaultDaemonDir is the default directory for daemon files.
const DefaultDaemonDir = ".bazelle"

// DefaultSocketName is the default socket file name.
const DefaultSocketName = "daemon.sock"

// DefaultPIDName is the default PID file name.
const DefaultPIDName = "daemon.pid"

// DefaultLogName is the default log file name.
const DefaultLogName = "daemon.log"

// Paths holds the paths for daemon files.
type Paths struct {
	Dir    string // directory containing daemon files
	Socket string // Unix socket path
	PID    string // PID file path
	Log    string // Log file path
}

// DefaultPaths returns the default daemon file paths.
func DefaultPaths() (*Paths, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	dir := filepath.Join(homeDir, DefaultDaemonDir)
	return &Paths{
		Dir:    dir,
		Socket: filepath.Join(dir, DefaultSocketName),
		PID:    filepath.Join(dir, DefaultPIDName),
		Log:    filepath.Join(dir, DefaultLogName),
	}, nil
}

// WorkspacePaths returns daemon file paths for a specific workspace.
func WorkspacePaths(workspaceRoot string) *Paths {
	dir := filepath.Join(workspaceRoot, DefaultDaemonDir)
	return &Paths{
		Dir:    dir,
		Socket: filepath.Join(dir, DefaultSocketName),
		PID:    filepath.Join(dir, DefaultPIDName),
		Log:    filepath.Join(dir, DefaultLogName),
	}
}

// EnsureDir ensures the daemon directory exists with proper permissions.
func (p *Paths) EnsureDir() error {
	return os.MkdirAll(p.Dir, 0700)
}

// WritePID writes the current process ID to the PID file.
func (p *Paths) WritePID() error {
	if err := p.EnsureDir(); err != nil {
		return err
	}
	pid := os.Getpid()
	return os.WriteFile(p.PID, []byte(strconv.Itoa(pid)), 0600)
}

// ReadPID reads the process ID from the PID file.
func (p *Paths) ReadPID() (int, error) {
	data, err := os.ReadFile(p.PID)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid PID file contents: %w", err)
	}
	return pid, nil
}

// RemovePID removes the PID file.
func (p *Paths) RemovePID() error {
	return os.Remove(p.PID)
}

// RemoveSocket removes the socket file.
func (p *Paths) RemoveSocket() error {
	return os.Remove(p.Socket)
}

// Cleanup removes all daemon files (PID, socket).
func (p *Paths) Cleanup() error {
	var errs []error
	if err := p.RemovePID(); err != nil && !os.IsNotExist(err) {
		errs = append(errs, fmt.Errorf("failed to remove PID file: %w", err))
	}
	if err := p.RemoveSocket(); err != nil && !os.IsNotExist(err) {
		errs = append(errs, fmt.Errorf("failed to remove socket: %w", err))
	}
	if len(errs) > 0 {
		return errs[0] // return first error
	}
	return nil
}

// IsProcessRunning checks if a process with the given PID is running.
func IsProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, FindProcess always succeeds. Send signal 0 to check if process exists.
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// DaemonStatus represents the current status of the daemon.
type DaemonStatus struct {
	Running    bool
	PID        int
	SocketPath string
	Stale      bool // true if PID file exists but process is not running
}

// GetStatus returns the current daemon status.
// If paths is nil, returns a status indicating the daemon is not running.
func GetStatus(paths *Paths) *DaemonStatus {
	if paths == nil {
		return &DaemonStatus{}
	}

	status := &DaemonStatus{
		SocketPath: paths.Socket,
	}

	pid, err := paths.ReadPID()
	if err != nil {
		// No PID file or invalid - daemon not running
		return status
	}

	status.PID = pid

	if IsProcessRunning(pid) {
		status.Running = true
	} else {
		// PID file exists but process is not running - stale
		status.Stale = true
	}

	return status
}

// CleanupStale removes stale daemon files if the daemon is not running.
// Returns true if cleanup was performed.
// If paths is nil, returns false with no error.
func CleanupStale(paths *Paths) (bool, error) {
	if paths == nil {
		return false, nil
	}
	status := GetStatus(paths)

	if status.Running {
		return false, nil
	}

	if !status.Stale && status.PID == 0 {
		// No stale files to clean
		// But check if socket file exists without PID file
		if _, err := os.Stat(paths.Socket); err == nil {
			// Orphan socket file exists - remove it
			if err := paths.RemoveSocket(); err != nil {
				return false, fmt.Errorf("failed to remove orphan socket: %w", err)
			}
			return true, nil
		}
		return false, nil
	}

	// Clean up stale files
	if err := paths.Cleanup(); err != nil {
		return false, err
	}

	return true, nil
}

// StopProcess sends SIGTERM to a process and returns whether it was stopped.
func StopProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process %d: %w", pid, err)
	}

	return process.Signal(syscall.SIGTERM)
}

// KillProcess sends SIGKILL to a process.
func KillProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process %d: %w", pid, err)
	}

	return process.Signal(syscall.SIGKILL)
}
