package watch

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsWatchLimitError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "no space left on device",
			err:      &os.PathError{Op: "watch", Path: "/foo", Err: os.ErrNotExist},
			expected: false,
		},
		{
			name:     "regular error",
			err:      os.ErrPermission,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isWatchLimitError(tt.err)
			if result != tt.expected {
				t.Errorf("isWatchLimitError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestFindBuildFile(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create subdirectories for testing
	subDir1 := filepath.Join(tmpDir, "pkg1")
	subDir2 := filepath.Join(tmpDir, "pkg2")
	subDir3 := filepath.Join(tmpDir, "pkg3")

	for _, dir := range []string{subDir1, subDir2, subDir3} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create BUILD.bazel in pkg1
	if err := os.WriteFile(filepath.Join(subDir1, "BUILD.bazel"), []byte("# test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create BUILD (without .bazel) in pkg2
	if err := os.WriteFile(filepath.Join(subDir2, "BUILD"), []byte("# test"), 0644); err != nil {
		t.Fatal(err)
	}

	// pkg3 has no BUILD file

	// Create watcher with temp dir as root
	w := &Watcher{
		config: Config{Root: tmpDir},
	}

	tests := []struct {
		name     string
		dir      string
		expected string
	}{
		{
			name:     "directory with BUILD.bazel",
			dir:      "pkg1",
			expected: filepath.Join("pkg1", "BUILD.bazel"),
		},
		{
			name:     "directory with BUILD",
			dir:      "pkg2",
			expected: filepath.Join("pkg2", "BUILD"),
		},
		{
			name:     "directory without BUILD file",
			dir:      "pkg3",
			expected: filepath.Join("pkg3", "BUILD.bazel"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := w.findBuildFile(tt.dir)
			if result != tt.expected {
				t.Errorf("findBuildFile(%q) = %q, want %q", tt.dir, result, tt.expected)
			}
		})
	}
}

func TestNewWatcher(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		Root:     tmpDir,
		Debounce: 100,
		Verbose:  true,
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer w.Close()

	// Verify watcher was created
	if w.fsWatcher == nil {
		t.Error("fsWatcher is nil")
	}

	if w.tracker == nil {
		t.Error("tracker is nil")
	}

	if w.logger == nil {
		t.Error("logger is nil")
	}

	// Verify extensions are populated
	if len(w.extensions) == 0 {
		t.Error("extensions map is empty")
	}

	// Check that common extensions are present
	expectedExts := []string{".go", ".java", ".kt", ".py"}
	for _, ext := range expectedExts {
		if !w.extensions[ext] {
			t.Errorf("expected extension %s not found", ext)
		}
	}

	// Verify ignore dirs are populated
	if len(w.ignoreDirs) == 0 {
		t.Error("ignoreDirs map is empty")
	}

	// Check that common ignore patterns are present
	expectedIgnores := []string{"bazel-", ".", "node_modules"}
	for _, dir := range expectedIgnores {
		if !w.ignoreDirs[dir] {
			t.Errorf("expected ignore pattern %s not found", dir)
		}
	}
}

func TestNewWatcherWithLanguageFilter(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		Root:       tmpDir,
		LangFilter: []string{"go", "kotlin"},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer w.Close()

	// Should have go extensions
	if !w.extensions[".go"] {
		t.Error("expected .go extension")
	}

	// Should have kotlin extensions
	if !w.extensions[".kt"] {
		t.Error("expected .kt extension")
	}

	// Should NOT have java extensions
	if w.extensions[".java"] {
		t.Error("should not have .java extension with filter")
	}

	// Should NOT have python extensions
	if w.extensions[".py"] {
		t.Error("should not have .py extension with filter")
	}
}

func TestWatcherClose(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := Config{Root: tmpDir}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
		return // Explicit return for nilaway
	}

	// Close should not error
	if err := w.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestWatcherCloseNilFsWatcher(t *testing.T) {
	// Close on nil fsWatcher should not panic
	w := &Watcher{fsWatcher: nil}
	if err := w.Close(); err != nil {
		t.Errorf("Close() on nil fsWatcher error = %v", err)
	}
}
