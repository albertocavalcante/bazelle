package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()

	// Check defaults
	if !cfg.IsLanguageEnabled("go") {
		t.Error("go should be enabled by default")
	}
	if !cfg.IsLanguageEnabled("proto") {
		t.Error("proto should be enabled by default")
	}
	if cfg.IsLanguageEnabled("kotlin") {
		t.Error("kotlin should be disabled by default")
	}
	if cfg.IsLanguageEnabled("python") {
		t.Error("python should be disabled by default")
	}

	// Check Go defaults
	if cfg.Go.NamingConvention != "import" {
		t.Errorf("go naming convention should be 'import', got %q", cfg.Go.NamingConvention)
	}

	// Check Kotlin defaults
	if cfg.Kotlin.LibraryMacro != "kt_jvm_library" {
		t.Errorf("kotlin library macro should be 'kt_jvm_library', got %q", cfg.Kotlin.LibraryMacro)
	}
}

func TestIsLanguageEnabled(t *testing.T) {
	cfg := NewConfig()

	// Test explicit enabled list
	cfg.Languages.Enabled = []string{"go", "kotlin", "python"}
	trueVal := true
	cfg.Kotlin.Enabled = &trueVal
	cfg.Python.Enabled = &trueVal

	if !cfg.IsLanguageEnabled("kotlin") {
		t.Error("kotlin should be enabled")
	}
	if !cfg.IsLanguageEnabled("python") {
		t.Error("python should be enabled")
	}

	// Test disabled takes precedence
	cfg.Languages.Disabled = []string{"kotlin"}
	if cfg.IsLanguageEnabled("kotlin") {
		t.Error("kotlin should be disabled when in disabled list")
	}
}

func TestGetEnabledLanguages(t *testing.T) {
	cfg := NewConfig()
	enabled := cfg.GetEnabledLanguages()

	// Default should have proto and go
	found := make(map[string]bool)
	for _, lang := range enabled {
		found[lang] = true
	}

	if !found["go"] {
		t.Error("go should be in enabled languages")
	}
	if !found["proto"] {
		t.Error("proto should be in enabled languages")
	}
	if found["kotlin"] {
		t.Error("kotlin should not be in enabled languages by default")
	}
}

func TestMerge(t *testing.T) {
	base := NewConfig()
	other := &Config{
		Languages: LanguagesConfig{
			Enabled: []string{"go", "kotlin", "python"},
		},
	}
	trueVal := true
	other.Kotlin.Enabled = &trueVal
	other.Python.Enabled = &trueVal
	other.Kotlin.ParserBackend = "treesitter"

	base.Merge(other)

	if !base.IsLanguageEnabled("kotlin") {
		t.Error("kotlin should be enabled after merge")
	}
	if base.Kotlin.ParserBackend != "treesitter" {
		t.Errorf("kotlin parser backend should be 'treesitter', got %q", base.Kotlin.ParserBackend)
	}
}

func TestLoadConfigFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	configContent := `
[languages]
enabled = ["go", "kotlin", "python"]
disabled = ["java"]

[kotlin]
enabled = true
parser_backend = "treesitter"

[python]
enabled = true
test_framework = "pytest"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg := loadConfigFile(configPath)
	if cfg == nil {
		t.Fatal("loadConfigFile returned nil")
	}

	// Check languages
	if len(cfg.Languages.Enabled) != 3 {
		t.Errorf("expected 3 enabled languages, got %d", len(cfg.Languages.Enabled))
	}
	if len(cfg.Languages.Disabled) != 1 {
		t.Errorf("expected 1 disabled language, got %d", len(cfg.Languages.Disabled))
	}

	// Check kotlin config
	if cfg.Kotlin.Enabled == nil || !*cfg.Kotlin.Enabled {
		t.Error("kotlin should be enabled")
	}
	if cfg.Kotlin.ParserBackend != "treesitter" {
		t.Errorf("kotlin parser backend should be 'treesitter', got %q", cfg.Kotlin.ParserBackend)
	}

	// Check python config
	if cfg.Python.Enabled == nil || !*cfg.Python.Enabled {
		t.Error("python should be enabled")
	}
	if cfg.Python.TestFramework != "pytest" {
		t.Errorf("python test framework should be 'pytest', got %q", cfg.Python.TestFramework)
	}
}

func TestApplyEnvironmentVariables(t *testing.T) {
	cfg := NewConfig()

	// Set environment variables
	t.Setenv("BAZELLE_LANGUAGES_ENABLED", "go,kotlin,python")
	t.Setenv("BAZELLE_KOTLIN_ENABLED", "true")
	t.Setenv("BAZELLE_KOTLIN_PARSER_BACKEND", "hybrid")

	applyEnvironmentVariables(cfg)

	if len(cfg.Languages.Enabled) != 3 {
		t.Errorf("expected 3 enabled languages, got %d", len(cfg.Languages.Enabled))
	}
	if cfg.Kotlin.Enabled == nil || !*cfg.Kotlin.Enabled {
		t.Error("kotlin should be enabled via env var")
	}
	if cfg.Kotlin.ParserBackend != "hybrid" {
		t.Errorf("kotlin parser backend should be 'hybrid', got %q", cfg.Kotlin.ParserBackend)
	}
}

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"go,kotlin,python", []string{"go", "kotlin", "python"}},
		{" go , kotlin , python ", []string{"go", "kotlin", "python"}},
		{"go", []string{"go"}},
		{"", []string{}},
		{" , , ", []string{}},
	}

	for _, tt := range tests {
		result := splitAndTrim(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("splitAndTrim(%q) = %v, want %v", tt.input, result, tt.expected)
			continue
		}
		for i, v := range result {
			if v != tt.expected[i] {
				t.Errorf("splitAndTrim(%q)[%d] = %q, want %q", tt.input, i, v, tt.expected[i])
			}
		}
	}
}

func TestProjectConfigSearch(t *testing.T) {
	// Create a temp directory structure
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project", "subdir")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Create .git marker at project root
	gitDir := filepath.Join(tmpDir, "project", ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}

	// Create bazelle.toml at project root
	configPath := filepath.Join(tmpDir, "project", "bazelle.toml")
	configContent := `
[languages]
enabled = ["go", "kotlin"]

[kotlin]
enabled = true
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Load config from subdir
	cfg := loadProjectConfigFrom(projectDir)
	if cfg == nil {
		t.Fatal("loadProjectConfigFrom returned nil")
	}

	if len(cfg.Languages.Enabled) != 2 {
		t.Errorf("expected 2 enabled languages, got %d", len(cfg.Languages.Enabled))
	}
}

func TestWorkspaceRootDetection(t *testing.T) {
	tmpDir := t.TempDir()

	// Test .git
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}
	if !isWorkspaceRoot(tmpDir) {
		t.Error("directory with .git should be workspace root")
	}

	// Test WORKSPACE
	tmpDir2 := t.TempDir()
	workspaceFile := filepath.Join(tmpDir2, "WORKSPACE")
	if err := os.WriteFile(workspaceFile, []byte(""), 0o644); err != nil {
		t.Fatalf("failed to write WORKSPACE file: %v", err)
	}
	if !isWorkspaceRoot(tmpDir2) {
		t.Error("directory with WORKSPACE should be workspace root")
	}

	// Test MODULE.bazel
	tmpDir3 := t.TempDir()
	moduleFile := filepath.Join(tmpDir3, "MODULE.bazel")
	if err := os.WriteFile(moduleFile, []byte(""), 0o644); err != nil {
		t.Fatalf("failed to write MODULE.bazel file: %v", err)
	}
	if !isWorkspaceRoot(tmpDir3) {
		t.Error("directory with MODULE.bazel should be workspace root")
	}
}
