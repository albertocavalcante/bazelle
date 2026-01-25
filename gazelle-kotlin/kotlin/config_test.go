package kotlin

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

func TestNewKotlinConfig(t *testing.T) {
	kc := NewKotlinConfig()

	// Test default values
	if kc.Enabled != false {
		t.Errorf("Expected Enabled to be false, got %v", kc.Enabled)
	}
	if kc.LibraryMacro != "kt_jvm_library" {
		t.Errorf("Expected LibraryMacro to be 'kt_jvm_library', got '%s'", kc.LibraryMacro)
	}
	if kc.TestMacro != "kt_jvm_test" {
		t.Errorf("Expected TestMacro to be 'kt_jvm_test', got '%s'", kc.TestMacro)
	}
	if kc.Visibility != "//visibility:public" {
		t.Errorf("Expected Visibility to be '//visibility:public', got '%s'", kc.Visibility)
	}
	if kc.LoadPath != "" {
		t.Errorf("Expected LoadPath to be empty, got '%s'", kc.LoadPath)
	}
}

func TestGetKotlinConfig_WithConfig(t *testing.T) {
	// Create a config with KotlinConfig
	c := &config.Config{
		Exts: make(map[string]interface{}),
	}
	kc := NewKotlinConfig()
	kc.Enabled = true
	kc.LibraryMacro = "custom_library"
	c.Exts[kotlinName] = kc

	// Get the config
	result := GetKotlinConfig(c)

	if result.Enabled != true {
		t.Errorf("Expected Enabled to be true, got %v", result.Enabled)
	}
	if result.LibraryMacro != "custom_library" {
		t.Errorf("Expected LibraryMacro to be 'custom_library', got '%s'", result.LibraryMacro)
	}
}

func TestGetKotlinConfig_NilConfig(t *testing.T) {
	// Create a config without KotlinConfig
	c := &config.Config{
		Exts: make(map[string]interface{}),
	}

	// Get the config - should return defaults
	result := GetKotlinConfig(c)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.Enabled != false {
		t.Errorf("Expected default Enabled to be false, got %v", result.Enabled)
	}
	if result.LibraryMacro != "kt_jvm_library" {
		t.Errorf("Expected default LibraryMacro to be 'kt_jvm_library', got '%s'", result.LibraryMacro)
	}
}

func TestGetKotlinConfig_WrongType(t *testing.T) {
	// Create a config with wrong type in Exts
	c := &config.Config{
		Exts: make(map[string]interface{}),
	}
	c.Exts[kotlinName] = "wrong type"

	// Get the config - should return defaults
	result := GetKotlinConfig(c)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.Enabled != false {
		t.Errorf("Expected default Enabled to be false, got %v", result.Enabled)
	}
}

func TestConfigure_KotlinEnabled(t *testing.T) {
	c := &config.Config{
		Exts: make(map[string]interface{}),
	}
	kc := NewKotlinConfig()
	c.Exts[kotlinName] = kc

	lang := &kotlinLang{}

	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"enabled true", "true", true},
		{"enabled TRUE", "TRUE", true},
		{"enabled True", "True", true},
		{"disabled false", "false", false},
		{"disabled FALSE", "FALSE", false},
		{"disabled other", "other", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset config
			c.Exts[kotlinName] = NewKotlinConfig()

			f := &rule.File{
				Directives: []rule.Directive{
					{Key: "kotlin_enabled", Value: tt.value},
				},
			}

			lang.Configure(c, "", f)

			result := GetKotlinConfig(c)
			if result.Enabled != tt.expected {
				t.Errorf("kotlin_enabled=%s: expected Enabled=%v, got %v", tt.value, tt.expected, result.Enabled)
			}
		})
	}
}

func TestConfigure_LibraryMacro(t *testing.T) {
	c := &config.Config{
		Exts: make(map[string]interface{}),
	}
	c.Exts[kotlinName] = NewKotlinConfig()

	lang := &kotlinLang{}
	f := &rule.File{
		Directives: []rule.Directive{
			{Key: "kotlin_library_macro", Value: "kt_library"},
		},
	}

	lang.Configure(c, "", f)

	result := GetKotlinConfig(c)
	if result.LibraryMacro != "kt_library" {
		t.Errorf("Expected LibraryMacro to be 'kt_library', got '%s'", result.LibraryMacro)
	}
}

func TestConfigure_TestMacro(t *testing.T) {
	c := &config.Config{
		Exts: make(map[string]interface{}),
	}
	c.Exts[kotlinName] = NewKotlinConfig()

	lang := &kotlinLang{}
	f := &rule.File{
		Directives: []rule.Directive{
			{Key: "kotlin_test_macro", Value: "kt_test"},
		},
	}

	lang.Configure(c, "", f)

	result := GetKotlinConfig(c)
	if result.TestMacro != "kt_test" {
		t.Errorf("Expected TestMacro to be 'kt_test', got '%s'", result.TestMacro)
	}
}

func TestConfigure_Visibility(t *testing.T) {
	c := &config.Config{
		Exts: make(map[string]interface{}),
	}
	c.Exts[kotlinName] = NewKotlinConfig()

	lang := &kotlinLang{}
	f := &rule.File{
		Directives: []rule.Directive{
			{Key: "kotlin_visibility", Value: "//visibility:private"},
		},
	}

	lang.Configure(c, "", f)

	result := GetKotlinConfig(c)
	if result.Visibility != "//visibility:private" {
		t.Errorf("Expected Visibility to be '//visibility:private', got '%s'", result.Visibility)
	}
}

func TestConfigure_LoadPath(t *testing.T) {
	c := &config.Config{
		Exts: make(map[string]interface{}),
	}
	c.Exts[kotlinName] = NewKotlinConfig()

	lang := &kotlinLang{}
	f := &rule.File{
		Directives: []rule.Directive{
			{Key: "kotlin_load", Value: "//build_defs:kotlin.bzl"},
		},
	}

	lang.Configure(c, "", f)

	result := GetKotlinConfig(c)
	if result.LoadPath != "//build_defs:kotlin.bzl" {
		t.Errorf("Expected LoadPath to be '//build_defs:kotlin.bzl', got '%s'", result.LoadPath)
	}
}

func TestConfigure_MultipleDirectives(t *testing.T) {
	c := &config.Config{
		Exts: make(map[string]interface{}),
	}
	c.Exts[kotlinName] = NewKotlinConfig()

	lang := &kotlinLang{}
	f := &rule.File{
		Directives: []rule.Directive{
			{Key: "kotlin_enabled", Value: "true"},
			{Key: "kotlin_library_macro", Value: "kt_lib"},
			{Key: "kotlin_test_macro", Value: "kt_tst"},
			{Key: "kotlin_visibility", Value: "//visibility:internal"},
			{Key: "kotlin_load", Value: "//defs:kotlin.bzl"},
		},
	}

	lang.Configure(c, "", f)

	result := GetKotlinConfig(c)
	if result.Enabled != true {
		t.Errorf("Expected Enabled to be true, got %v", result.Enabled)
	}
	if result.LibraryMacro != "kt_lib" {
		t.Errorf("Expected LibraryMacro to be 'kt_lib', got '%s'", result.LibraryMacro)
	}
	if result.TestMacro != "kt_tst" {
		t.Errorf("Expected TestMacro to be 'kt_tst', got '%s'", result.TestMacro)
	}
	if result.Visibility != "//visibility:internal" {
		t.Errorf("Expected Visibility to be '//visibility:internal', got '%s'", result.Visibility)
	}
	if result.LoadPath != "//defs:kotlin.bzl" {
		t.Errorf("Expected LoadPath to be '//defs:kotlin.bzl', got '%s'", result.LoadPath)
	}
}

func TestConfigure_NilFile(t *testing.T) {
	c := &config.Config{
		Exts: make(map[string]interface{}),
	}
	kc := NewKotlinConfig()
	kc.Enabled = true
	c.Exts[kotlinName] = kc

	lang := &kotlinLang{}

	// Should not panic with nil file
	lang.Configure(c, "", nil)

	// Config should be preserved
	result := GetKotlinConfig(c)
	if result.Enabled != true {
		t.Errorf("Expected Enabled to remain true, got %v", result.Enabled)
	}
}

func TestConfigure_InheritanceFromParent(t *testing.T) {
	// Set up parent config
	c := &config.Config{
		Exts: make(map[string]interface{}),
	}
	parentKc := NewKotlinConfig()
	parentKc.Enabled = true
	parentKc.LibraryMacro = "parent_lib"
	c.Exts[kotlinName] = parentKc

	lang := &kotlinLang{}

	// Configure with empty file (should inherit parent config)
	f := &rule.File{
		Directives: []rule.Directive{},
	}

	lang.Configure(c, "", f)

	result := GetKotlinConfig(c)
	// Should inherit parent settings
	if result.Enabled != true {
		t.Errorf("Expected to inherit Enabled=true from parent, got %v", result.Enabled)
	}
	if result.LibraryMacro != "parent_lib" {
		t.Errorf("Expected to inherit LibraryMacro='parent_lib' from parent, got '%s'", result.LibraryMacro)
	}
}

func TestConfigure_OverrideParent(t *testing.T) {
	// Set up parent config
	c := &config.Config{
		Exts: make(map[string]interface{}),
	}
	parentKc := NewKotlinConfig()
	parentKc.Enabled = true
	parentKc.LibraryMacro = "parent_lib"
	c.Exts[kotlinName] = parentKc

	lang := &kotlinLang{}

	// Configure with directive that overrides parent
	f := &rule.File{
		Directives: []rule.Directive{
			{Key: "kotlin_library_macro", Value: "child_lib"},
		},
	}

	lang.Configure(c, "", f)

	result := GetKotlinConfig(c)
	// Should inherit Enabled but override LibraryMacro
	if result.Enabled != true {
		t.Errorf("Expected to inherit Enabled=true from parent, got %v", result.Enabled)
	}
	if result.LibraryMacro != "child_lib" {
		t.Errorf("Expected LibraryMacro to be overridden to 'child_lib', got '%s'", result.LibraryMacro)
	}
}

func TestConfigure_ParserBackend(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected ParserBackendType
	}{
		{"heuristic", "heuristic", BackendHeuristic},
		{"treesitter", "treesitter", BackendTreeSitter},
		{"hybrid", "hybrid", BackendHybrid},
		{"TREESITTER uppercase", "TREESITTER", BackendTreeSitter},
		{"Hybrid mixed case", "Hybrid", BackendHybrid},
		{"invalid defaults to heuristic", "invalid", BackendHeuristic},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &config.Config{
				Exts: make(map[string]interface{}),
			}
			c.Exts[kotlinName] = NewKotlinConfig()

			lang := &kotlinLang{}
			f := &rule.File{
				Directives: []rule.Directive{
					{Key: "kotlin_parser_backend", Value: tt.value},
				},
			}

			lang.Configure(c, "", f)

			result := GetKotlinConfig(c)
			if result.ParserBackend != tt.expected {
				t.Errorf("kotlin_parser_backend=%s: expected %v, got %v", tt.value, tt.expected, result.ParserBackend)
			}
		})
	}
}

func TestConfigure_FQNScanning(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"enabled true", "true", true},
		{"enabled TRUE", "TRUE", true},
		{"disabled false", "false", false},
		{"disabled other", "other", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &config.Config{
				Exts: make(map[string]interface{}),
			}
			c.Exts[kotlinName] = NewKotlinConfig()

			lang := &kotlinLang{}
			f := &rule.File{
				Directives: []rule.Directive{
					{Key: "kotlin_fqn_scanning", Value: tt.value},
				},
			}

			lang.Configure(c, "", f)

			result := GetKotlinConfig(c)
			if result.EnableFQNScanning != tt.expected {
				t.Errorf("kotlin_fqn_scanning=%s: expected %v, got %v", tt.value, tt.expected, result.EnableFQNScanning)
			}
		})
	}
}

func TestKnownDirectives(t *testing.T) {
	lang := &kotlinLang{}
	directives := lang.KnownDirectives()

	expected := []string{
		"kotlin_enabled",
		"kotlin_library_macro",
		"kotlin_test_macro",
		"kotlin_visibility",
		"kotlin_load",
		"kotlin_parser_backend",
		"kotlin_fqn_scanning",
	}

	if len(directives) != len(expected) {
		t.Fatalf("Expected %d directives, got %d: %v", len(expected), len(directives), directives)
	}

	directiveSet := make(map[string]bool)
	for _, d := range directives {
		directiveSet[d] = true
	}

	for _, exp := range expected {
		if !directiveSet[exp] {
			t.Errorf("Expected directive '%s' not found", exp)
		}
	}
}
