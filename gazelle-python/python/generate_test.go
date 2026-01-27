package python

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/rule"
)

// ============================================================================
// deriveTargetName Tests
// ============================================================================

func TestDeriveTargetName(t *testing.T) {
	tests := []struct {
		name     string
		dir      string
		repoRoot string
		expected string
	}{
		{
			name:     "simple_directory",
			dir:      "/home/user/repo/mypackage",
			repoRoot: "/home/user/repo",
			expected: "mypackage",
		},
		{
			name:     "nested_directory",
			dir:      "/home/user/repo/src/mypackage",
			repoRoot: "/home/user/repo",
			expected: "mypackage",
		},
		{
			name:     "directory_with_hyphen",
			dir:      "/home/user/repo/my-package",
			repoRoot: "/home/user/repo",
			expected: "my_package",
		},
		{
			name:     "root_directory",
			dir:      "/home/user/repo",
			repoRoot: "/home/user/repo",
			expected: "lib",
		},
		{
			name:     "deeply_nested",
			dir:      "/home/user/repo/a/b/c/d/mymod",
			repoRoot: "/home/user/repo",
			expected: "mymod",
		},
		{
			name:     "multiple_hyphens",
			dir:      "/home/user/repo/my-cool-package",
			repoRoot: "/home/user/repo",
			expected: "my_cool_package",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deriveTargetName(tt.dir, tt.repoRoot)
			if result != tt.expected {
				t.Errorf("deriveTargetName(%q, %q) = %q, want %q",
					tt.dir, tt.repoRoot, result, tt.expected)
			}
		})
	}
}

// ============================================================================
// getSrcGlobs Tests
// ============================================================================

func TestGetSrcGlobs(t *testing.T) {
	result := getSrcGlobs()

	// Check it returns a GlobValue
	if len(result.Patterns) == 0 {
		t.Error("expected non-empty patterns")
	}

	// Should include *.py pattern
	foundPy := false
	for _, p := range result.Patterns {
		if p == "*.py" {
			foundPy = true
		}
	}
	if !foundPy {
		t.Errorf("expected '*.py' in patterns, got %v", result.Patterns)
	}

	// Should exclude test files
	foundTestExclude := false
	for _, e := range result.Excludes {
		if e == "*_test.py" || e == "test_*.py" {
			foundTestExclude = true
		}
	}
	if !foundTestExclude {
		t.Errorf("expected test file exclusions, got %v", result.Excludes)
	}
}

// ============================================================================
// getTestSrcGlobs Tests
// ============================================================================

func TestGetTestSrcGlobs(t *testing.T) {
	result := getTestSrcGlobs()

	// Check it returns a GlobValue
	if len(result.Patterns) == 0 {
		t.Error("expected non-empty patterns")
	}

	// Should include test patterns
	foundTestSuffix := false
	foundTestPrefix := false
	for _, p := range result.Patterns {
		if p == "*_test.py" {
			foundTestSuffix = true
		}
		if p == "test_*.py" {
			foundTestPrefix = true
		}
	}
	if !foundTestSuffix {
		t.Errorf("expected '*_test.py' in patterns, got %v", result.Patterns)
	}
	if !foundTestPrefix {
		t.Errorf("expected 'test_*.py' in patterns, got %v", result.Patterns)
	}
}

// ============================================================================
// findPythonSources Tests
// ============================================================================

func TestFindPythonSources(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()

	// Create Python source files
	files := map[string]string{
		"main.py":         "# main",
		"utils.py":        "# utils",
		"test_main.py":    "# test main",
		"utils_test.py":   "# utils test",
		"__init__.py":     "# init",
		"not_python.txt":  "text file",
		"conftest.py":     "# conftest",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to create %s: %v", name, err)
		}
	}

	t.Run("main_sources", func(t *testing.T) {
		sources := findPythonSources(tmpDir, false)

		// Should find main.py, utils.py, conftest.py
		// Should NOT find __init__.py, test files, or non-python files
		expected := map[string]bool{
			"main.py":     true,
			"utils.py":    true,
			"conftest.py": true,
		}

		if len(sources) != len(expected) {
			t.Errorf("expected %d main sources, got %d: %v", len(expected), len(sources), sources)
		}

		for _, src := range sources {
			if !expected[src] {
				t.Errorf("unexpected source file: %s", src)
			}
		}
	})

	t.Run("test_sources", func(t *testing.T) {
		sources := findPythonSources(tmpDir, true)

		// Should find test_main.py, utils_test.py
		// Should NOT find main sources or __init__.py
		expected := map[string]bool{
			"test_main.py":  true,
			"utils_test.py": true,
		}

		if len(sources) != len(expected) {
			t.Errorf("expected %d test sources, got %d: %v", len(expected), len(sources), sources)
		}

		for _, src := range sources {
			if !expected[src] {
				t.Errorf("unexpected test source file: %s", src)
			}
		}
	})
}

func TestFindPythonSourcesEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	sources := findPythonSources(tmpDir, false)
	if len(sources) != 0 {
		t.Errorf("expected no sources in empty dir, got %v", sources)
	}

	testSources := findPythonSources(tmpDir, true)
	if len(testSources) != 0 {
		t.Errorf("expected no test sources in empty dir, got %v", testSources)
	}
}

func TestFindPythonSourcesNonExistentDir(t *testing.T) {
	sources := findPythonSources("/nonexistent/path/that/does/not/exist", false)
	if sources != nil && len(sources) != 0 {
		t.Errorf("expected nil or empty for nonexistent dir, got %v", sources)
	}
}

func TestFindPythonSourcesOnlyInit(t *testing.T) {
	tmpDir := t.TempDir()

	// Create only __init__.py
	initPath := filepath.Join(tmpDir, "__init__.py")
	if err := os.WriteFile(initPath, []byte("# init"), 0o644); err != nil {
		t.Fatalf("failed to create __init__.py: %v", err)
	}

	sources := findPythonSources(tmpDir, false)
	if len(sources) != 0 {
		t.Errorf("expected no sources (__init__.py should be excluded), got %v", sources)
	}
}

// ============================================================================
// GlobValue Tests
// ============================================================================

func TestGlobValueStructure(t *testing.T) {
	// Test that GlobValue has the expected fields
	gv := rule.GlobValue{
		Patterns: []string{"*.py"},
		Excludes: []string{"*_test.py"},
	}

	if len(gv.Patterns) != 1 {
		t.Errorf("expected 1 pattern, got %d", len(gv.Patterns))
	}
	if len(gv.Excludes) != 1 {
		t.Errorf("expected 1 exclude, got %d", len(gv.Excludes))
	}
}

// ============================================================================
// Integration Tests for generate.go helper functions
// ============================================================================

func TestFindPythonSourcesWithSubdirectory(t *testing.T) {
	// Create a temp dir with subdirectories
	tmpDir := t.TempDir()

	// Create files in root
	if err := os.WriteFile(filepath.Join(tmpDir, "main.py"), []byte("# main"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a subdirectory with files (these should NOT be included)
	subDir := filepath.Join(tmpDir, "subpackage")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "sub.py"), []byte("# sub"), 0o644); err != nil {
		t.Fatal(err)
	}

	sources := findPythonSources(tmpDir, false)

	// Should only find main.py, not sub.py (subdirectories are not searched)
	if len(sources) != 1 {
		t.Errorf("expected 1 source, got %d: %v", len(sources), sources)
	}
	if len(sources) > 0 && sources[0] != "main.py" {
		t.Errorf("expected main.py, got %v", sources)
	}
}

func TestDeriveTargetNameInvalidRelPath(t *testing.T) {
	// Test when filepath.Rel would fail (different drives on Windows, etc.)
	// This is hard to test portably, but we can test the fallback
	result := deriveTargetName("/some/path/package", "/some/path")
	if result != "package" {
		t.Errorf("expected 'package', got %q", result)
	}
}
