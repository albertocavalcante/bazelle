package python

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// ============================================================================
// PythonConfig Tests
// ============================================================================

func TestNewPythonConfig(t *testing.T) {
	pc := NewPythonConfig()

	if pc.Enabled {
		t.Error("expected Enabled to be false by default")
	}
	if pc.LibraryMacro != "py_library" {
		t.Errorf("expected LibraryMacro to be 'py_library', got %q", pc.LibraryMacro)
	}
	if pc.TestMacro != "py_test" {
		t.Errorf("expected TestMacro to be 'py_test', got %q", pc.TestMacro)
	}
	if pc.BinaryMacro != "py_binary" {
		t.Errorf("expected BinaryMacro to be 'py_binary', got %q", pc.BinaryMacro)
	}
	if pc.Visibility != "//visibility:public" {
		t.Errorf("expected Visibility to be '//visibility:public', got %q", pc.Visibility)
	}
	if pc.TestFramework != "pytest" {
		t.Errorf("expected TestFramework to be 'pytest', got %q", pc.TestFramework)
	}
}

func TestPythonConfigClone(t *testing.T) {
	original := &PythonConfig{
		Enabled:           true,
		LibraryMacro:      "custom_py_library",
		TestMacro:         "custom_py_test",
		BinaryMacro:       "custom_py_binary",
		Visibility:        "//visibility:private",
		LoadPath:          "//custom:defs.bzl",
		TestFramework:     "unittest",
		StdlibModulesFile: "/path/to/stdlib.txt",
	}

	clone := original.Clone()

	// Verify all fields are copied
	if clone.Enabled != original.Enabled {
		t.Error("Enabled not cloned correctly")
	}
	if clone.LibraryMacro != original.LibraryMacro {
		t.Error("LibraryMacro not cloned correctly")
	}
	if clone.TestMacro != original.TestMacro {
		t.Error("TestMacro not cloned correctly")
	}
	if clone.BinaryMacro != original.BinaryMacro {
		t.Error("BinaryMacro not cloned correctly")
	}
	if clone.Visibility != original.Visibility {
		t.Error("Visibility not cloned correctly")
	}
	if clone.LoadPath != original.LoadPath {
		t.Error("LoadPath not cloned correctly")
	}
	if clone.TestFramework != original.TestFramework {
		t.Error("TestFramework not cloned correctly")
	}
	if clone.StdlibModulesFile != original.StdlibModulesFile {
		t.Error("StdlibModulesFile not cloned correctly")
	}

	// Verify it's a true copy (modifying clone doesn't affect original)
	clone.Enabled = false
	if original.Enabled != true {
		t.Error("clone modification affected original")
	}
}

func TestGetPythonConfigWithNil(t *testing.T) {
	c := &config.Config{
		Exts: make(map[string]interface{}),
	}

	// When no config exists, should return default
	pc := GetPythonConfig(c)
	if pc == nil {
		t.Fatal("expected non-nil config")
	}
	if pc.LibraryMacro != "py_library" {
		t.Error("expected default config values")
	}
}

func TestGetPythonConfigWithExisting(t *testing.T) {
	c := &config.Config{
		Exts: make(map[string]interface{}),
	}
	existingConfig := &PythonConfig{
		Enabled:      true,
		LibraryMacro: "custom_macro",
	}
	c.Exts[pythonName] = existingConfig

	pc := GetPythonConfig(c)
	if pc != existingConfig {
		t.Error("expected to get existing config")
	}
	if pc.LibraryMacro != "custom_macro" {
		t.Error("expected custom macro from existing config")
	}
}

func TestGetPythonConfigWithWrongType(t *testing.T) {
	c := &config.Config{
		Exts: make(map[string]interface{}),
	}
	// Set something that's not a *PythonConfig
	c.Exts[pythonName] = "wrong type"

	pc := GetPythonConfig(c)
	if pc == nil {
		t.Fatal("expected non-nil config")
	}
	// Should return default config when type assertion fails
	if pc.LibraryMacro != "py_library" {
		t.Error("expected default config values when type is wrong")
	}
}

// ============================================================================
// pythonLang Config Methods Tests
// ============================================================================

func TestKnownDirectives(t *testing.T) {
	lang := &pythonLang{}
	directives := lang.KnownDirectives()

	expected := []string{
		"python_enabled",
		"python_library_macro",
		"python_test_macro",
		"python_binary_macro",
		"python_visibility",
		"python_load",
		"python_test_framework",
		"python_stdlib_modules_file",
	}

	if len(directives) != len(expected) {
		t.Errorf("expected %d directives, got %d", len(expected), len(directives))
	}

	directiveSet := make(map[string]bool)
	for _, d := range directives {
		directiveSet[d] = true
	}

	for _, exp := range expected {
		if !directiveSet[exp] {
			t.Errorf("expected directive %q not found", exp)
		}
	}
}

func TestConfigureWithNilFile(t *testing.T) {
	lang := &pythonLang{}
	c := &config.Config{
		Exts: make(map[string]interface{}),
	}
	c.Exts[pythonName] = NewPythonConfig()

	// Should not panic with nil file
	lang.Configure(c, "", nil)

	pc := GetPythonConfig(c)
	if pc == nil {
		t.Error("expected config to exist after Configure")
	}
}

func TestConfigureWithDirectives(t *testing.T) {
	lang := &pythonLang{}
	c := &config.Config{
		Exts: make(map[string]interface{}),
	}
	c.Exts[pythonName] = NewPythonConfig()

	f := &rule.File{
		Directives: []rule.Directive{
			{Key: "python_enabled", Value: "true"},
			{Key: "python_library_macro", Value: "my_py_library"},
			{Key: "python_test_macro", Value: "my_py_test"},
			{Key: "python_binary_macro", Value: "my_py_binary"},
			{Key: "python_visibility", Value: "//visibility:private"},
			{Key: "python_load", Value: "//my:defs.bzl"},
			{Key: "python_test_framework", Value: "unittest"},
			{Key: "python_stdlib_modules_file", Value: "/path/to/modules.txt"},
		},
	}

	lang.Configure(c, "some/rel/path", f)

	pc := GetPythonConfig(c)
	if pc == nil {
		t.Fatal("expected config to exist")
	}

	if !pc.Enabled {
		t.Error("expected Enabled to be true")
	}
	if pc.LibraryMacro != "my_py_library" {
		t.Errorf("expected LibraryMacro 'my_py_library', got %q", pc.LibraryMacro)
	}
	if pc.TestMacro != "my_py_test" {
		t.Errorf("expected TestMacro 'my_py_test', got %q", pc.TestMacro)
	}
	if pc.BinaryMacro != "my_py_binary" {
		t.Errorf("expected BinaryMacro 'my_py_binary', got %q", pc.BinaryMacro)
	}
	if pc.Visibility != "//visibility:private" {
		t.Errorf("expected Visibility '//visibility:private', got %q", pc.Visibility)
	}
	if pc.LoadPath != "//my:defs.bzl" {
		t.Errorf("expected LoadPath '//my:defs.bzl', got %q", pc.LoadPath)
	}
	if pc.TestFramework != "unittest" {
		t.Errorf("expected TestFramework 'unittest', got %q", pc.TestFramework)
	}
	if pc.StdlibModulesFile != "/path/to/modules.txt" {
		t.Errorf("expected StdlibModulesFile '/path/to/modules.txt', got %q", pc.StdlibModulesFile)
	}
}

func TestConfigureWithInvalidTestFramework(t *testing.T) {
	lang := &pythonLang{}
	c := &config.Config{
		Exts: make(map[string]interface{}),
	}
	c.Exts[pythonName] = NewPythonConfig()

	f := &rule.File{
		Directives: []rule.Directive{
			{Key: "python_test_framework", Value: "invalid_framework"},
		},
	}

	lang.Configure(c, "", f)

	pc := GetPythonConfig(c)
	// Should default to pytest when invalid framework is specified
	if pc.TestFramework != "pytest" {
		t.Errorf("expected pytest for invalid framework, got %q", pc.TestFramework)
	}
}

func TestConfigureEnabledCaseInsensitive(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"true", true},
		{"True", true},
		{"TRUE", true},
		{"false", false},
		{"False", false},
		{"anything_else", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			lang := &pythonLang{}
			c := &config.Config{
				Exts: make(map[string]interface{}),
			}
			c.Exts[pythonName] = NewPythonConfig()

			f := &rule.File{
				Directives: []rule.Directive{
					{Key: "python_enabled", Value: tt.value},
				},
			}

			lang.Configure(c, "", f)

			pc := GetPythonConfig(c)
			if pc.Enabled != tt.expected {
				t.Errorf("python_enabled=%q: got Enabled=%v, want %v",
					tt.value, pc.Enabled, tt.expected)
			}
		})
	}
}

func TestConfigureTestFrameworkCaseInsensitive(t *testing.T) {
	tests := []struct {
		value    string
		expected string
	}{
		{"pytest", "pytest"},
		{"Pytest", "pytest"},
		{"PYTEST", "pytest"},
		{"unittest", "unittest"},
		{"Unittest", "unittest"},
		{"UNITTEST", "unittest"},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			lang := &pythonLang{}
			c := &config.Config{
				Exts: make(map[string]interface{}),
			}
			c.Exts[pythonName] = NewPythonConfig()

			f := &rule.File{
				Directives: []rule.Directive{
					{Key: "python_test_framework", Value: tt.value},
				},
			}

			lang.Configure(c, "", f)

			pc := GetPythonConfig(c)
			if pc.TestFramework != tt.expected {
				t.Errorf("python_test_framework=%q: got %q, want %q",
					tt.value, pc.TestFramework, tt.expected)
			}
		})
	}
}

func TestConfigureWithNoExistingConfig(t *testing.T) {
	lang := &pythonLang{}
	c := &config.Config{
		Exts: make(map[string]interface{}),
	}
	// Don't set any config - Configure should create one

	f := &rule.File{
		Directives: []rule.Directive{
			{Key: "python_enabled", Value: "true"},
		},
	}

	lang.Configure(c, "", f)

	pc := GetPythonConfig(c)
	if pc == nil {
		t.Fatal("expected config to be created")
	}
	if !pc.Enabled {
		t.Error("expected Enabled to be true")
	}
}
