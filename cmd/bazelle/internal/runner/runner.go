// Package runner provides functionality to find and execute the gazelle binary.
package runner

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// ErrGazelleNotFound is returned when the gazelle binary cannot be located.
var ErrGazelleNotFound = errors.New("gazelle binary not found")

// Runner handles finding and executing the gazelle binary.
type Runner struct {
	executablePath string // Path to bazelle executable (for finding sibling)
}

// Option configures a Runner.
type Option func(*Runner)

// WithExecutablePath sets the path to the bazelle executable.
// Used primarily for testing.
func WithExecutablePath(path string) Option {
	return func(r *Runner) {
		r.executablePath = path
	}
}

// New creates a new Runner with the given options.
func New(opts ...Option) *Runner {
	r := &Runner{}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// FindGazelleBinary locates the gazelle binary using the following search order:
// 1. Bazel runfiles (when run via `bazel run`)
// 2. Sibling binary (bazelle-gazelle next to bazelle)
// 3. PATH lookup
func (r *Runner) FindGazelleBinary() (string, error) {
	exe := r.executablePath
	if exe == "" {
		var err error
		exe, err = os.Executable()
		if err != nil {
			return "", fmt.Errorf("failed to get executable path: %w", err)
		}
	}

	// 1. Check runfiles (bazel run)
	if path := r.findInRunfiles(exe); path != "" {
		return path, nil
	}

	// 2. Check sibling binary
	if path := r.findSibling(exe); path != "" {
		return path, nil
	}

	// 3. Check PATH
	if path, err := exec.LookPath("bazelle-gazelle"); err == nil {
		return path, nil
	}

	return "", ErrGazelleNotFound
}

// findInRunfiles looks for the gazelle binary in bazel runfiles.
func (r *Runner) findInRunfiles(exe string) string {
	// Standard runfiles location: <binary>.runfiles/<workspace>/<path>
	runfilesDir := exe + ".runfiles"
	candidates := []string{
		// Bazel bzlmod style: _main/<path>/<target>_/<binary>
		filepath.Join(runfilesDir, "_main", "cmd", "gazelle", "gazelle_", "gazelle"),
		filepath.Join(runfilesDir, "_main", "cmd", "gazelle", "gazelle"),
		// Legacy workspace style
		filepath.Join(runfilesDir, "bazelle", "cmd", "gazelle", "gazelle_", "gazelle"),
		filepath.Join(runfilesDir, "bazelle", "cmd", "gazelle", "gazelle"),
	}

	for _, candidate := range candidates {
		if fileExists(candidate) {
			return candidate
		}
	}
	return ""
}

// findSibling looks for bazelle-gazelle next to the bazelle binary.
func (r *Runner) findSibling(exe string) string {
	dir := filepath.Dir(exe)
	sibling := filepath.Join(dir, "bazelle-gazelle")
	if fileExists(sibling) {
		return sibling
	}
	return ""
}

// Exec executes the gazelle binary with the given arguments.
// This replaces the current process (unix exec).
func (r *Runner) Exec(args []string) error {
	gazellePath, err := r.FindGazelleBinary()
	if err != nil {
		return err
	}

	// Prepend the binary path to args (argv[0])
	argv := append([]string{gazellePath}, args...)

	// Replace current process with gazelle
	return syscall.Exec(gazellePath, argv, os.Environ())
}

// Run executes the gazelle binary and returns after it completes.
// Use this when you need to capture output or continue after gazelle runs.
func (r *Runner) Run(args []string) error {
	gazellePath, err := r.FindGazelleBinary()
	if err != nil {
		return err
	}

	cmd := exec.Command(gazellePath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// RunWithOutput executes gazelle and captures its output.
func (r *Runner) RunWithOutput(args []string) ([]byte, error) {
	gazellePath, err := r.FindGazelleBinary()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(gazellePath, args...)
	return cmd.CombinedOutput()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
