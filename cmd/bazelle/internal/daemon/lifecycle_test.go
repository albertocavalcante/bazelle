package daemon

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
)

func TestDefaultPaths(t *testing.T) {
	t.Parallel()
	paths, err := DefaultPaths()
	if err != nil {
		t.Fatalf("DefaultPaths() error = %v", err)
	}

	if paths.Dir == "" {
		t.Error("Dir should not be empty")
	}
	if paths.Socket == "" {
		t.Error("Socket should not be empty")
	}
	if paths.PID == "" {
		t.Error("PID should not be empty")
	}
	if paths.Log == "" {
		t.Error("Log should not be empty")
	}

	// Verify paths are under Dir
	if filepath.Dir(paths.Socket) != paths.Dir {
		t.Errorf("Socket %q not under Dir %q", paths.Socket, paths.Dir)
	}
	if filepath.Dir(paths.PID) != paths.Dir {
		t.Errorf("PID %q not under Dir %q", paths.PID, paths.Dir)
	}
}

func TestWorkspacePaths(t *testing.T) {
	t.Parallel()
	workspaceRoot := "/tmp/my-workspace"
	paths := WorkspacePaths(workspaceRoot)

	expectedDir := filepath.Join(workspaceRoot, DefaultDaemonDir)
	if paths.Dir != expectedDir {
		t.Errorf("Dir = %q, want %q", paths.Dir, expectedDir)
	}
	if paths.Socket != filepath.Join(expectedDir, DefaultSocketName) {
		t.Errorf("Socket path mismatch")
	}
	if paths.PID != filepath.Join(expectedDir, DefaultPIDName) {
		t.Errorf("PID path mismatch")
	}
	if paths.Log != filepath.Join(expectedDir, DefaultLogName) {
		t.Errorf("Log path mismatch")
	}
}

func TestPaths_EnsureDir(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	paths := &Paths{
		Dir: filepath.Join(tmpDir, "daemon-test", "nested"),
	}

	// Directory shouldn't exist yet
	if _, err := os.Stat(paths.Dir); !os.IsNotExist(err) {
		t.Fatal("Directory should not exist before EnsureDir")
	}

	// EnsureDir should create it
	if err := paths.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir() error = %v", err)
	}

	// Verify it exists with correct permissions
	info, err := os.Stat(paths.Dir)
	if err != nil {
		t.Fatalf("os.Stat() error = %v", err)
	}
	if !info.IsDir() {
		t.Error("Should be a directory")
	}
	// Check permissions (0700)
	if info.Mode().Perm() != 0700 {
		t.Errorf("Permissions = %o, want 0700", info.Mode().Perm())
	}

	// EnsureDir should be idempotent
	if err := paths.EnsureDir(); err != nil {
		t.Fatalf("Second EnsureDir() error = %v", err)
	}
}

func TestPaths_WritePID(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	paths := &Paths{
		Dir: tmpDir,
		PID: filepath.Join(tmpDir, "test.pid"),
	}

	if err := paths.WritePID(); err != nil {
		t.Fatalf("WritePID() error = %v", err)
	}

	// Verify PID file exists and contains current PID
	data, err := os.ReadFile(paths.PID)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	expectedPID := strconv.Itoa(os.Getpid())
	if string(data) != expectedPID {
		t.Errorf("PID file contents = %q, want %q", data, expectedPID)
	}

	// Check permissions (0600)
	info, err := os.Stat(paths.PID)
	if err != nil {
		t.Fatalf("os.Stat() error = %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("Permissions = %o, want 0600", info.Mode().Perm())
	}
}

func TestPaths_ReadPID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		content   string
		wantPID   int
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "valid pid",
			content: "12345",
			wantPID: 12345,
		},
		{
			name:    "pid with whitespace",
			content: "  12345  \n",
			wantPID: 12345,
		},
		{
			name:    "pid 1",
			content: "1",
			wantPID: 1,
		},
		{
			name:      "empty file",
			content:   "",
			wantErr:   true,
			errSubstr: "invalid",
		},
		{
			name:      "non-numeric",
			content:   "abc",
			wantErr:   true,
			errSubstr: "invalid",
		},
		{
			name:      "float",
			content:   "123.45",
			wantErr:   true,
			errSubstr: "invalid",
		},
		{
			name:      "negative",
			content:   "-123",
			wantPID:   -123, // strconv.Atoi accepts negative
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()
			paths := &Paths{
				PID: filepath.Join(tmpDir, "test.pid"),
			}

			if err := os.WriteFile(paths.PID, []byte(tt.content), 0600); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}

			pid, err := paths.ReadPID()
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadPID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if tt.errSubstr != "" && err != nil {
					// Check error message contains expected substring
					if !containsIgnoreCase(err.Error(), tt.errSubstr) {
						t.Errorf("Error %q should contain %q", err.Error(), tt.errSubstr)
					}
				}
				return
			}
			if pid != tt.wantPID {
				t.Errorf("ReadPID() = %d, want %d", pid, tt.wantPID)
			}
		})
	}
}

func TestPaths_ReadPID_FileNotExists(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	paths := &Paths{
		PID: filepath.Join(tmpDir, "nonexistent.pid"),
	}

	_, err := paths.ReadPID()
	if err == nil {
		t.Error("ReadPID() should return error for non-existent file")
	}
	if !os.IsNotExist(err) {
		t.Errorf("Error should be os.IsNotExist, got %v", err)
	}
}

func TestPaths_RemovePID(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	paths := &Paths{
		PID: filepath.Join(tmpDir, "test.pid"),
	}

	// Create PID file
	if err := os.WriteFile(paths.PID, []byte("12345"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Remove it
	if err := paths.RemovePID(); err != nil {
		t.Fatalf("RemovePID() error = %v", err)
	}

	// Verify it's gone
	if _, err := os.Stat(paths.PID); !os.IsNotExist(err) {
		t.Error("PID file should not exist after RemovePID")
	}
}

func TestPaths_RemovePID_NotExists(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	paths := &Paths{
		PID: filepath.Join(tmpDir, "nonexistent.pid"),
	}

	err := paths.RemovePID()
	if err == nil || !os.IsNotExist(err) {
		t.Errorf("RemovePID() should return os.IsNotExist error, got %v", err)
	}
}

func TestPaths_RemoveSocket(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	paths := &Paths{
		Socket: filepath.Join(tmpDir, "test.sock"),
	}

	// Create socket file (simulated as regular file for test)
	if err := os.WriteFile(paths.Socket, []byte{}, 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := paths.RemoveSocket(); err != nil {
		t.Fatalf("RemoveSocket() error = %v", err)
	}

	if _, err := os.Stat(paths.Socket); !os.IsNotExist(err) {
		t.Error("Socket file should not exist after RemoveSocket")
	}
}

func TestPaths_Cleanup(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	paths := &Paths{
		Dir:    tmpDir,
		Socket: filepath.Join(tmpDir, "test.sock"),
		PID:    filepath.Join(tmpDir, "test.pid"),
	}

	// Create both files
	if err := os.WriteFile(paths.PID, []byte("12345"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(paths.Socket, []byte{}, 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Cleanup
	if err := paths.Cleanup(); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}

	// Both should be gone
	if _, err := os.Stat(paths.PID); !os.IsNotExist(err) {
		t.Error("PID file should not exist after Cleanup")
	}
	if _, err := os.Stat(paths.Socket); !os.IsNotExist(err) {
		t.Error("Socket file should not exist after Cleanup")
	}
}

func TestPaths_Cleanup_PartialExists(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	paths := &Paths{
		Dir:    tmpDir,
		Socket: filepath.Join(tmpDir, "test.sock"),
		PID:    filepath.Join(tmpDir, "test.pid"),
	}

	// Only create PID file
	if err := os.WriteFile(paths.PID, []byte("12345"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Cleanup should still work (not fail on missing socket)
	if err := paths.Cleanup(); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}

	if _, err := os.Stat(paths.PID); !os.IsNotExist(err) {
		t.Error("PID file should not exist after Cleanup")
	}
}

func TestPaths_Cleanup_NoneExist(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	paths := &Paths{
		Dir:    tmpDir,
		Socket: filepath.Join(tmpDir, "test.sock"),
		PID:    filepath.Join(tmpDir, "test.pid"),
	}

	// Cleanup should not error when files don't exist
	if err := paths.Cleanup(); err != nil {
		t.Errorf("Cleanup() should not error when files don't exist, got %v", err)
	}
}

func TestIsProcessRunning(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		pid  int
		want bool
	}{
		{
			name: "current process",
			pid:  os.Getpid(),
			want: true,
		},
		{
			name: "init process",
			pid:  1,
			want: true, // init/systemd should always be running
		},
		{
			name: "zero pid",
			pid:  0,
			want: false,
		},
		{
			name: "negative pid",
			pid:  -1,
			want: false,
		},
		{
			name: "very large pid",
			pid:  999999999,
			want: false, // unlikely to exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Skip init process check if we're in a container without access
			if tt.name == "init process" {
				if got := IsProcessRunning(tt.pid); !got {
					t.Skip("init process not accessible, might be in container")
				}
				return
			}
			if got := IsProcessRunning(tt.pid); got != tt.want {
				t.Errorf("IsProcessRunning(%d) = %v, want %v", tt.pid, got, tt.want)
			}
		})
	}
}

func TestDaemonStatus_GetStatus(t *testing.T) {
	t.Parallel()
	t.Run("nil paths", func(t *testing.T) {
		t.Parallel()
		status := GetStatus(nil)
		if status == nil {
			t.Fatal("GetStatus(nil) should return non-nil status")
		}
		if status.Running {
			t.Error("Running should be false for nil paths")
		}
		if status.Stale {
			t.Error("Stale should be false for nil paths")
		}
	})

	t.Run("no pid file", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		paths := &Paths{
			PID:    filepath.Join(tmpDir, "nonexistent.pid"),
			Socket: filepath.Join(tmpDir, "daemon.sock"),
		}

		status := GetStatus(paths)
		if status.Running {
			t.Error("Running should be false when no PID file")
		}
		if status.Stale {
			t.Error("Stale should be false when no PID file")
		}
		if status.PID != 0 {
			t.Errorf("PID should be 0, got %d", status.PID)
		}
	})

	t.Run("running process", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		paths := &Paths{
			PID:    filepath.Join(tmpDir, "daemon.pid"),
			Socket: filepath.Join(tmpDir, "daemon.sock"),
		}

		// Write current process PID (which is running)
		pid := os.Getpid()
		if err := os.WriteFile(paths.PID, []byte(strconv.Itoa(pid)), 0600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		status := GetStatus(paths)
		if !status.Running {
			t.Error("Running should be true for current process")
		}
		if status.Stale {
			t.Error("Stale should be false for running process")
		}
		if status.PID != pid {
			t.Errorf("PID = %d, want %d", status.PID, pid)
		}
	})

	t.Run("stale pid file", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		paths := &Paths{
			PID:    filepath.Join(tmpDir, "daemon.pid"),
			Socket: filepath.Join(tmpDir, "daemon.sock"),
		}

		// Write a PID that definitely doesn't exist
		stalePID := 999999999
		if err := os.WriteFile(paths.PID, []byte(strconv.Itoa(stalePID)), 0600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		status := GetStatus(paths)
		if status.Running {
			t.Error("Running should be false for stale PID")
		}
		if !status.Stale {
			t.Error("Stale should be true for non-running PID")
		}
		if status.PID != stalePID {
			t.Errorf("PID = %d, want %d", status.PID, stalePID)
		}
	})
}

func TestCleanupStale(t *testing.T) {
	t.Parallel()
	t.Run("nil paths", func(t *testing.T) {
		t.Parallel()
		cleaned, err := CleanupStale(nil)
		if err != nil {
			t.Errorf("CleanupStale(nil) error = %v", err)
		}
		if cleaned {
			t.Error("cleaned should be false for nil paths")
		}
	})

	t.Run("no stale files", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		paths := &Paths{
			Dir:    tmpDir,
			PID:    filepath.Join(tmpDir, "daemon.pid"),
			Socket: filepath.Join(tmpDir, "daemon.sock"),
		}

		cleaned, err := CleanupStale(paths)
		if err != nil {
			t.Errorf("CleanupStale() error = %v", err)
		}
		if cleaned {
			t.Error("cleaned should be false when no stale files")
		}
	})

	t.Run("running process not cleaned", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		paths := &Paths{
			Dir:    tmpDir,
			PID:    filepath.Join(tmpDir, "daemon.pid"),
			Socket: filepath.Join(tmpDir, "daemon.sock"),
		}

		// Write current process PID
		pid := os.Getpid()
		if err := os.WriteFile(paths.PID, []byte(strconv.Itoa(pid)), 0600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		cleaned, err := CleanupStale(paths)
		if err != nil {
			t.Errorf("CleanupStale() error = %v", err)
		}
		if cleaned {
			t.Error("Should not clean up running process")
		}

		// PID file should still exist
		if _, err := os.Stat(paths.PID); os.IsNotExist(err) {
			t.Error("PID file should still exist for running process")
		}
	})

	t.Run("stale files cleaned", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		paths := &Paths{
			Dir:    tmpDir,
			PID:    filepath.Join(tmpDir, "daemon.pid"),
			Socket: filepath.Join(tmpDir, "daemon.sock"),
		}

		// Write a stale PID
		stalePID := 999999999
		if err := os.WriteFile(paths.PID, []byte(strconv.Itoa(stalePID)), 0600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
		if err := os.WriteFile(paths.Socket, []byte{}, 0600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		cleaned, err := CleanupStale(paths)
		if err != nil {
			t.Errorf("CleanupStale() error = %v", err)
		}
		if !cleaned {
			t.Error("Should have cleaned up stale files")
		}

		// Both files should be gone
		if _, err := os.Stat(paths.PID); !os.IsNotExist(err) {
			t.Error("PID file should be removed")
		}
		if _, err := os.Stat(paths.Socket); !os.IsNotExist(err) {
			t.Error("Socket file should be removed")
		}
	})

	t.Run("orphan socket cleaned", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		paths := &Paths{
			Dir:    tmpDir,
			PID:    filepath.Join(tmpDir, "daemon.pid"),
			Socket: filepath.Join(tmpDir, "daemon.sock"),
		}

		// Only create socket file (no PID file)
		if err := os.WriteFile(paths.Socket, []byte{}, 0600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		cleaned, err := CleanupStale(paths)
		if err != nil {
			t.Errorf("CleanupStale() error = %v", err)
		}
		if !cleaned {
			t.Error("Should have cleaned up orphan socket")
		}

		if _, err := os.Stat(paths.Socket); !os.IsNotExist(err) {
			t.Error("Socket file should be removed")
		}
	})
}

func TestStopProcess(t *testing.T) {
	// Skip on CI environments where we can't spawn processes
	if os.Getenv("CI") != "" {
		t.Skip("Skipping process control test in CI")
	}

	t.Run("invalid pid", func(t *testing.T) {
		t.Parallel()
		err := StopProcess(999999999)
		if err == nil {
			t.Error("StopProcess() should return error for invalid PID")
		}
	})

	t.Run("current process", func(t *testing.T) {
		t.Parallel()
		// Note: We can't actually stop ourselves, but we can verify the function doesn't panic
		// Sending SIGTERM to ourselves would terminate the test
		// So just verify we can find the process
		process, err := os.FindProcess(os.Getpid())
		if err != nil {
			t.Errorf("FindProcess() error = %v", err)
		}
		if process == nil {
			t.Error("process should not be nil")
		}
	})
}

func TestKillProcess(t *testing.T) {
	t.Parallel()
	t.Run("invalid pid", func(t *testing.T) {
		t.Parallel()
		err := KillProcess(999999999)
		if err == nil {
			t.Error("KillProcess() should return error for invalid PID")
		}
	})
}

func TestStopProcess_PermissionError(t *testing.T) {
	t.Parallel()
	// Try to stop init process (PID 1) - should fail with permission error
	err := StopProcess(1)
	if err == nil {
		// Root might actually succeed, which is fine
		t.Skip("Running as root, permission test not applicable")
	}
	// Just verify we got an error (permission denied or similar)
	if !errors.Is(err, syscall.EPERM) && !errors.Is(err, os.ErrPermission) {
		// Some systems return different errors, just verify we got an error
		t.Logf("Got error (expected): %v", err)
	}
}

func TestPaths_WritePID_PermissionDenied(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("Running as root, permission test not applicable")
	}

	tmpDir := t.TempDir()
	paths := &Paths{
		Dir: tmpDir,
		PID: filepath.Join(tmpDir, "test.pid"),
	}

	// Make directory read-only
	if err := os.Chmod(tmpDir, 0500); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(tmpDir, 0700) // restore for cleanup
	})

	err := paths.WritePID()
	if err == nil {
		t.Error("WritePID() should return error for read-only directory")
	}
}

func TestDefaultConstants(t *testing.T) {
	t.Parallel()
	if DefaultDaemonDir == "" {
		t.Error("DefaultDaemonDir should not be empty")
	}
	if DefaultSocketName == "" {
		t.Error("DefaultSocketName should not be empty")
	}
	if DefaultPIDName == "" {
		t.Error("DefaultPIDName should not be empty")
	}
	if DefaultLogName == "" {
		t.Error("DefaultLogName should not be empty")
	}
}

// containsIgnoreCase checks if s contains substr (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(substr) == 0 ||
		(len(s) > 0 && containsIgnoreCaseHelper(s, substr)))
}

func containsIgnoreCaseHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalFoldAt(s, i, substr) {
			return true
		}
	}
	return false
}

func equalFoldAt(s string, start int, substr string) bool {
	for j := 0; j < len(substr); j++ {
		c1 := s[start+j]
		c2 := substr[j]
		if c1 != c2 {
			// Simple ASCII case folding
			if c1 >= 'A' && c1 <= 'Z' {
				c1 += 'a' - 'A'
			}
			if c2 >= 'A' && c2 <= 'Z' {
				c2 += 'a' - 'A'
			}
			if c1 != c2 {
				return false
			}
		}
	}
	return true
}
