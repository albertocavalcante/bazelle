package jvm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindSourceFiles(t *testing.T) {
	// Create a temporary directory structure
	tmpDir, err := os.MkdirTemp("", "jvm_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create source files
	srcDir := filepath.Join(tmpDir, "src", "main", "kotlin")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}

	files := []string{"Main.kt", "Helper.kt", "Script.kts", "README.md"}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(srcDir, f), []byte(""), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", f, err)
		}
	}

	// Test finding Kotlin files
	found := FindSourceFiles(tmpDir, "src/main/kotlin", []string{".kt", ".kts"})
	if len(found) != 3 {
		t.Errorf("FindSourceFiles() found %d files, want 3", len(found))
	}

	// Test with non-existent directory
	notFound := FindSourceFiles(tmpDir, "src/main/java", []string{".java"})
	if len(notFound) != 0 {
		t.Errorf("FindSourceFiles() found %d files in non-existent dir, want 0", len(notFound))
	}
}

func TestFindLanguageFiles(t *testing.T) {
	// Create a temporary directory structure
	tmpDir, err := os.MkdirTemp("", "jvm_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create Groovy source files
	srcDir := filepath.Join(tmpDir, "src", "test", "groovy")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}

	files := []string{"TestSpec.groovy", "HelperTest.groovy"}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(srcDir, f), []byte(""), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", f, err)
		}
	}

	found := FindLanguageFiles(tmpDir, "src/test/groovy", Groovy)
	if len(found) != 2 {
		t.Errorf("FindLanguageFiles() found %d files, want 2", len(found))
	}
}

func TestFindMainSources(t *testing.T) {
	// Create a temporary directory structure
	tmpDir, err := os.MkdirTemp("", "jvm_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create main source files
	srcDir := filepath.Join(tmpDir, "src", "main", "java")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(srcDir, "Main.java"), []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	found := FindMainSources(tmpDir, Java)
	if len(found) != 1 {
		t.Errorf("FindMainSources() found %d files, want 1", len(found))
	}
}

func TestFindTestSources(t *testing.T) {
	// Create a temporary directory structure
	tmpDir, err := os.MkdirTemp("", "jvm_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test source files
	srcDir := filepath.Join(tmpDir, "src", "test", "scala")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(srcDir, "MainTest.scala"), []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	found := FindTestSources(tmpDir, Scala)
	if len(found) != 1 {
		t.Errorf("FindTestSources() found %d files, want 1", len(found))
	}
}

func TestIsSourceDir(t *testing.T) {
	tests := []struct {
		dir  string
		lang Language
		want bool
	}{
		{"/project/src/main/kotlin/com/example", Kotlin, true},
		{"/project/src/test/kotlin/com/example", Kotlin, true},
		{"/project/src/main/java/com/example", Kotlin, false},
		{"/project/lib/kotlin", Kotlin, false},
	}

	for _, tt := range tests {
		t.Run(tt.dir, func(t *testing.T) {
			got := IsSourceDir(tt.dir, tt.lang)
			if got != tt.want {
				t.Errorf("IsSourceDir(%q, %v) = %v, want %v", tt.dir, tt.lang, got, tt.want)
			}
		})
	}
}

func TestIsTestDir(t *testing.T) {
	tests := []struct {
		dir  string
		want bool
	}{
		{"/project/src/test/kotlin", true},
		{"/project/src/test/java", true},
		{"/project/src/main/kotlin", false},
		{"/project/test/kotlin", false},
	}

	for _, tt := range tests {
		t.Run(tt.dir, func(t *testing.T) {
			got := IsTestDir(tt.dir)
			if got != tt.want {
				t.Errorf("IsTestDir(%q) = %v, want %v", tt.dir, got, tt.want)
			}
		})
	}
}

func TestIsMainDir(t *testing.T) {
	tests := []struct {
		dir  string
		want bool
	}{
		{"/project/src/main/kotlin", true},
		{"/project/src/main/java", true},
		{"/project/src/test/kotlin", false},
		{"/project/main/kotlin", false},
	}

	for _, tt := range tests {
		t.Run(tt.dir, func(t *testing.T) {
			got := IsMainDir(tt.dir)
			if got != tt.want {
				t.Errorf("IsMainDir(%q) = %v, want %v", tt.dir, got, tt.want)
			}
		})
	}
}
