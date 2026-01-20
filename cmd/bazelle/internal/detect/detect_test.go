package detect_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/albertocavalcante/bazelle/cmd/bazelle/internal/detect"
)

func TestDetectLanguages_Empty(t *testing.T) {
	tmpDir := t.TempDir()

	langs, err := detect.Languages(tmpDir)
	if err != nil {
		t.Fatalf("Languages() error = %v", err)
	}
	if len(langs) != 0 {
		t.Errorf("Languages() = %v, want empty", langs)
	}
}

func TestDetectLanguages_Go(t *testing.T) {
	tmpDir := t.TempDir()
	createFile(t, tmpDir, "main.go")
	createFile(t, tmpDir, "pkg/util.go")

	langs, err := detect.Languages(tmpDir)
	if err != nil {
		t.Fatalf("Languages() error = %v", err)
	}
	if !slices.Contains(langs, "go") {
		t.Errorf("Languages() = %v, want to contain 'go'", langs)
	}
}

func TestDetectLanguages_Kotlin(t *testing.T) {
	tmpDir := t.TempDir()
	createFile(t, tmpDir, "src/main/kotlin/Main.kt")
	createFile(t, tmpDir, "src/test/kotlin/MainTest.kt")

	langs, err := detect.Languages(tmpDir)
	if err != nil {
		t.Fatalf("Languages() error = %v", err)
	}
	if !slices.Contains(langs, "kotlin") {
		t.Errorf("Languages() = %v, want to contain 'kotlin'", langs)
	}
}

func TestDetectLanguages_Python(t *testing.T) {
	tmpDir := t.TempDir()
	createFile(t, tmpDir, "app.py")
	createFile(t, tmpDir, "tests/test_app.py")

	langs, err := detect.Languages(tmpDir)
	if err != nil {
		t.Fatalf("Languages() error = %v", err)
	}
	if !slices.Contains(langs, "python") {
		t.Errorf("Languages() = %v, want to contain 'python'", langs)
	}
}

func TestDetectLanguages_Proto(t *testing.T) {
	tmpDir := t.TempDir()
	createFile(t, tmpDir, "api/service.proto")

	langs, err := detect.Languages(tmpDir)
	if err != nil {
		t.Fatalf("Languages() error = %v", err)
	}
	if !slices.Contains(langs, "proto") {
		t.Errorf("Languages() = %v, want to contain 'proto'", langs)
	}
}

func TestDetectLanguages_CC(t *testing.T) {
	tmpDir := t.TempDir()
	createFile(t, tmpDir, "src/main.cc")
	createFile(t, tmpDir, "include/lib.h")

	langs, err := detect.Languages(tmpDir)
	if err != nil {
		t.Fatalf("Languages() error = %v", err)
	}
	if !slices.Contains(langs, "cc") {
		t.Errorf("Languages() = %v, want to contain 'cc'", langs)
	}
}

func TestDetectLanguages_Multiple(t *testing.T) {
	tmpDir := t.TempDir()
	createFile(t, tmpDir, "main.go")
	createFile(t, tmpDir, "app.py")
	createFile(t, tmpDir, "lib/Util.kt")

	langs, err := detect.Languages(tmpDir)
	if err != nil {
		t.Fatalf("Languages() error = %v", err)
	}

	want := []string{"go", "kotlin", "python"}
	for _, w := range want {
		if !slices.Contains(langs, w) {
			t.Errorf("Languages() = %v, missing %q", langs, w)
		}
	}
}

func TestDetectLanguages_IgnoresBazelDirs(t *testing.T) {
	tmpDir := t.TempDir()
	// Files in bazel-* dirs should be ignored
	createFile(t, tmpDir, "bazel-out/main.go")
	createFile(t, tmpDir, "bazel-bin/app.py")

	langs, err := detect.Languages(tmpDir)
	if err != nil {
		t.Fatalf("Languages() error = %v", err)
	}
	if len(langs) != 0 {
		t.Errorf("Languages() = %v, want empty (bazel dirs should be ignored)", langs)
	}
}

func TestDetectLanguages_IgnoresHiddenDirs(t *testing.T) {
	tmpDir := t.TempDir()
	createFile(t, tmpDir, ".git/hooks/pre-commit.py")
	createFile(t, tmpDir, ".idea/main.go")

	langs, err := detect.Languages(tmpDir)
	if err != nil {
		t.Fatalf("Languages() error = %v", err)
	}
	if len(langs) != 0 {
		t.Errorf("Languages() = %v, want empty (hidden dirs should be ignored)", langs)
	}
}

func createFile(t *testing.T, base, path string) {
	t.Helper()
	fullPath := filepath.Join(base, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte("// content"), 0o644); err != nil {
		t.Fatal(err)
	}
}
