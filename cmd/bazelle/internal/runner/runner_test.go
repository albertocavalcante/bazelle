package runner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/albertocavalcante/bazelle/cmd/bazelle/internal/runner"
)

func TestFindGazelleBinary_SiblingBinary(t *testing.T) {
	// Create temp directory with fake binaries
	tmpDir := t.TempDir()

	bazellePath := filepath.Join(tmpDir, "bazelle")
	gazellePath := filepath.Join(tmpDir, "bazelle-gazelle")

	if err := os.WriteFile(bazellePath, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(gazellePath, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}

	r := runner.New(runner.WithExecutablePath(bazellePath))
	got, err := r.FindGazelleBinary()
	if err != nil {
		t.Fatalf("FindGazelleBinary() error = %v", err)
	}
	if got != gazellePath {
		t.Errorf("FindGazelleBinary() = %q, want %q", got, gazellePath)
	}
}

func TestFindGazelleBinary_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	bazellePath := filepath.Join(tmpDir, "bazelle")

	if err := os.WriteFile(bazellePath, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}

	r := runner.New(runner.WithExecutablePath(bazellePath))
	_, err := r.FindGazelleBinary()
	if err == nil {
		t.Error("FindGazelleBinary() expected error for missing binary")
	}
}

func TestFindGazelleBinary_RunfilesPath(t *testing.T) {
	// Create temp directory simulating runfiles structure
	tmpDir := t.TempDir()

	bazellePath := filepath.Join(tmpDir, "bazelle")
	runfilesDir := filepath.Join(tmpDir, "bazelle.runfiles")
	gazelleRunfile := filepath.Join(runfilesDir, "bazelle", "cmd", "gazelle", "gazelle")

	if err := os.WriteFile(bazellePath, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(gazelleRunfile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(gazelleRunfile, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}

	r := runner.New(runner.WithExecutablePath(bazellePath))
	got, err := r.FindGazelleBinary()
	if err != nil {
		t.Fatalf("FindGazelleBinary() error = %v", err)
	}
	if got != gazelleRunfile {
		t.Errorf("FindGazelleBinary() = %q, want %q", got, gazelleRunfile)
	}
}
