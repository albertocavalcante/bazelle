package kotlin

import (
	"flag"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// KotlinConfig holds configuration for the Kotlin extension.
type KotlinConfig struct {
	// Enabled indicates whether the Kotlin extension is enabled.
	Enabled bool

	// LibraryMacro is the rule kind to use for libraries (default: kt_jvm_library).
	LibraryMacro string

	// TestMacro is the rule kind to use for tests (default: kt_jvm_test).
	TestMacro string

	// Visibility is the default visibility for generated targets.
	Visibility string

	// LoadPath is the path to load custom macros from.
	LoadPath string
}

// NewKotlinConfig creates a new KotlinConfig with default values.
func NewKotlinConfig() *KotlinConfig {
	return &KotlinConfig{
		Enabled:      false,
		LibraryMacro: "kt_jvm_library",
		TestMacro:    "kt_jvm_test",
		Visibility:   "//visibility:public",
		LoadPath:     "",
	}
}

// GetKotlinConfig extracts KotlinConfig from the Gazelle config.
func GetKotlinConfig(c *config.Config) *KotlinConfig {
	kc, ok := c.Exts[kotlinName].(*KotlinConfig)
	if !ok {
		return NewKotlinConfig()
	}
	return kc
}

// RegisterFlags implements config.Configurer.
func (*kotlinLang) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	kc := NewKotlinConfig()
	c.Exts[kotlinName] = kc
}

// CheckFlags implements config.Configurer.
func (*kotlinLang) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	return nil
}

// KnownDirectives implements config.Configurer.
func (*kotlinLang) KnownDirectives() []string {
	return []string{
		"kotlin_enabled",
		"kotlin_library_macro",
		"kotlin_test_macro",
		"kotlin_visibility",
		"kotlin_load",
	}
}

// Configure implements config.Configurer.
func (*kotlinLang) Configure(c *config.Config, rel string, f *rule.File) {
	kc := GetKotlinConfig(c)
	if kc == nil {
		kc = NewKotlinConfig()
		c.Exts[kotlinName] = kc
	}

	// Create a new config for this directory (inheriting from parent)
	newKc := *kc
	c.Exts[kotlinName] = &newKc

	if f == nil {
		return
	}

	for _, d := range f.Directives {
		switch d.Key {
		case "kotlin_enabled":
			newKc.Enabled = strings.ToLower(d.Value) == "true"
		case "kotlin_library_macro":
			newKc.LibraryMacro = d.Value
		case "kotlin_test_macro":
			newKc.TestMacro = d.Value
		case "kotlin_visibility":
			newKc.Visibility = d.Value
		case "kotlin_load":
			newKc.LoadPath = d.Value
		}
	}
}

// IsKotlinSourceDir checks if a directory contains Kotlin source files.
func IsKotlinSourceDir(dir string) bool {
	// Standard Maven/Gradle layout
	return strings.Contains(dir, filepath.Join("src", "main", "kotlin")) ||
		strings.Contains(dir, filepath.Join("src", "test", "kotlin"))
}

// IsTestDir checks if a directory is a test directory.
func IsTestDir(dir string) bool {
	return strings.Contains(dir, filepath.Join("src", "test"))
}
