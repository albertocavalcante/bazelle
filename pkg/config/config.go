// Package config provides configuration management for Bazelle.
// It supports multi-layer configuration with precedence:
//  1. Built-in defaults (lowest priority)
//  2. Global user config (~/.config/bazelle/config.toml)
//  3. Project config (.bazelle/config.toml or bazelle.toml)
//  4. Environment variables (BAZELLE_*)
//  5. CLI flags (highest priority)
package config

import "slices"

// Config is the main configuration struct for Bazelle.
type Config struct {
	// Languages configures which language extensions to enable.
	Languages LanguagesConfig `toml:"languages"`

	// Go configures the Go language extension.
	Go GoConfig `toml:"go"`

	// Kotlin configures the Kotlin language extension.
	Kotlin KotlinConfig `toml:"kotlin"`

	// Python configures the Python language extension.
	Python PythonConfig `toml:"python"`

	// Java configures the Java language extension.
	Java JavaConfig `toml:"java"`

	// Scala configures the Scala language extension.
	Scala ScalaConfig `toml:"scala"`

	// Groovy configures the Groovy language extension.
	Groovy GroovyConfig `toml:"groovy"`

	// Proto configures the Protocol Buffers extension.
	Proto ProtoConfig `toml:"proto"`

	// Rust configures the Rust language extension.
	Rust RustConfig `toml:"rust"`

	// CC configures the C/C++ language extension.
	CC CCConfig `toml:"cc"`

	// Bzl configures the Bazel Starlark language extension.
	Bzl BzlConfig `toml:"bzl"`
}

// LanguagesConfig specifies which languages to enable/disable.
type LanguagesConfig struct {
	// Enabled is the list of languages to enable (e.g., ["go", "kotlin", "python"]).
	// If empty, defaults to built-in defaults.
	Enabled []string `toml:"enabled"`

	// Disabled is the list of languages to explicitly disable.
	// Takes precedence over Enabled.
	Disabled []string `toml:"disabled"`
}

// GoConfig holds Go-specific configuration.
type GoConfig struct {
	// Enabled specifies whether the Go extension is enabled.
	Enabled *bool `toml:"enabled"`

	// NamingConvention is the Go naming convention ("import" or "go_default_library").
	NamingConvention string `toml:"naming_convention"`

	// NamingConventionExternal is the naming convention for external deps.
	NamingConventionExternal string `toml:"naming_convention_external"`
}

// KotlinConfig holds Kotlin-specific configuration.
type KotlinConfig struct {
	// Enabled specifies whether the Kotlin extension is enabled.
	Enabled *bool `toml:"enabled"`

	// ParserBackend is the parsing strategy ("heuristic", "treesitter", "hybrid").
	ParserBackend string `toml:"parser_backend"`

	// LibraryMacro is the macro to use for kt_jvm_library rules.
	LibraryMacro string `toml:"library_macro"`

	// TestMacro is the macro to use for kt_jvm_test rules.
	TestMacro string `toml:"test_macro"`

	// FQNScanning enables detection of fully-qualified names in code body.
	FQNScanning *bool `toml:"fqn_scanning"`
}

// PythonConfig holds Python-specific configuration.
type PythonConfig struct {
	// Enabled specifies whether the Python extension is enabled.
	Enabled *bool `toml:"enabled"`

	// StdlibModulesFile is an optional path to a custom stdlib modules list.
	StdlibModulesFile string `toml:"stdlib_modules_file"`

	// TestFramework is the test framework ("pytest", "unittest").
	TestFramework string `toml:"test_framework"`

	// LibraryMacro is the macro to use for py_library rules.
	LibraryMacro string `toml:"library_macro"`

	// TestMacro is the macro to use for py_test rules.
	TestMacro string `toml:"test_macro"`

	// BinaryMacro is the macro to use for py_binary rules.
	BinaryMacro string `toml:"binary_macro"`
}

// JavaConfig holds Java-specific configuration.
type JavaConfig struct {
	// Enabled specifies whether the Java extension is enabled.
	Enabled *bool `toml:"enabled"`

	// LibraryMacro is the macro to use for java_library rules.
	LibraryMacro string `toml:"library_macro"`

	// TestMacro is the macro to use for java_test rules.
	TestMacro string `toml:"test_macro"`
}

// ScalaConfig holds Scala-specific configuration.
type ScalaConfig struct {
	// Enabled specifies whether the Scala extension is enabled.
	Enabled *bool `toml:"enabled"`

	// LibraryMacro is the macro to use for scala_library rules.
	LibraryMacro string `toml:"library_macro"`

	// TestMacro is the macro to use for scala_test rules.
	TestMacro string `toml:"test_macro"`
}

// GroovyConfig holds Groovy-specific configuration.
type GroovyConfig struct {
	// Enabled specifies whether the Groovy extension is enabled.
	Enabled *bool `toml:"enabled"`

	// LibraryMacro is the macro to use for groovy_library rules.
	LibraryMacro string `toml:"library_macro"`

	// TestMacro is the macro to use for groovy_test rules.
	TestMacro string `toml:"test_macro"`

	// SpockTestMacro is the macro to use for Spock tests.
	SpockTestMacro string `toml:"spock_test_macro"`
}

// ProtoConfig holds Protocol Buffers configuration.
type ProtoConfig struct {
	// Enabled specifies whether the Proto extension is enabled.
	Enabled *bool `toml:"enabled"`
}

// RustConfig holds Rust-specific configuration.
type RustConfig struct {
	// Enabled specifies whether the Rust extension is enabled.
	Enabled *bool `toml:"enabled"`
}

// CCConfig holds C/C++ configuration.
type CCConfig struct {
	// Enabled specifies whether the C/C++ extension is enabled.
	Enabled *bool `toml:"enabled"`
}

// BzlConfig holds Bazel Starlark configuration.
type BzlConfig struct {
	// Enabled specifies whether the Bzl extension is enabled.
	Enabled *bool `toml:"enabled"`
}

// NewConfig creates a new Config with built-in defaults.
// By default, only Go and Proto are enabled (safest defaults).
func NewConfig() *Config {
	trueVal := true
	falseVal := false
	return &Config{
		Languages: LanguagesConfig{
			// Default: Go and Proto are always safe to enable
			Enabled:  []string{"go", "proto"},
			Disabled: []string{},
		},
		Go: GoConfig{
			Enabled:                  &trueVal,
			NamingConvention:         "import",
			NamingConventionExternal: "import",
		},
		Kotlin: KotlinConfig{
			Enabled:       &falseVal,
			ParserBackend: "heuristic",
			LibraryMacro:  "kt_jvm_library",
			TestMacro:     "kt_jvm_test",
			FQNScanning:   &trueVal,
		},
		Python: PythonConfig{
			Enabled:       &falseVal,
			TestFramework: "pytest",
			LibraryMacro:  "py_library",
			TestMacro:     "py_test",
			BinaryMacro:   "py_binary",
		},
		Java: JavaConfig{
			Enabled:      &falseVal,
			LibraryMacro: "java_library",
			TestMacro:    "java_test",
		},
		Scala: ScalaConfig{
			Enabled:      &falseVal,
			LibraryMacro: "scala_library",
			TestMacro:    "scala_test",
		},
		Groovy: GroovyConfig{
			Enabled:        &falseVal,
			LibraryMacro:   "groovy_library",
			TestMacro:      "groovy_test",
			SpockTestMacro: "spock_test",
		},
		Proto: ProtoConfig{
			Enabled: &trueVal,
		},
		Rust: RustConfig{
			Enabled: &falseVal,
		},
		CC: CCConfig{
			Enabled: &falseVal,
		},
		Bzl: BzlConfig{
			Enabled: &falseVal,
		},
	}
}

// IsLanguageEnabled checks if a language is enabled in the configuration.
func (c *Config) IsLanguageEnabled(lang string) bool {
	// Check explicit disabled list first (highest priority)
	if slices.Contains(c.Languages.Disabled, lang) {
		return false
	}

	// Check language-specific enabled flag
	switch lang {
	case "go":
		return c.Go.Enabled != nil && *c.Go.Enabled
	case "kotlin":
		return c.Kotlin.Enabled != nil && *c.Kotlin.Enabled
	case "python":
		return c.Python.Enabled != nil && *c.Python.Enabled
	case "java":
		return c.Java.Enabled != nil && *c.Java.Enabled
	case "scala":
		return c.Scala.Enabled != nil && *c.Scala.Enabled
	case "groovy":
		return c.Groovy.Enabled != nil && *c.Groovy.Enabled
	case "proto":
		return c.Proto.Enabled != nil && *c.Proto.Enabled
	case "rust":
		return c.Rust.Enabled != nil && *c.Rust.Enabled
	case "cc":
		return c.CC.Enabled != nil && *c.CC.Enabled
	case "bzl":
		return c.Bzl.Enabled != nil && *c.Bzl.Enabled
	}

	// Check if in enabled list
	return slices.Contains(c.Languages.Enabled, lang)
}

// GetEnabledLanguages returns the list of enabled language names.
func (c *Config) GetEnabledLanguages() []string {
	// Standard order: proto first, then others
	allLangs := []string{"proto", "go", "bzl", "java", "scala", "python", "cc", "kotlin", "groovy", "rust"}
	var enabled []string
	for _, lang := range allLangs {
		if c.IsLanguageEnabled(lang) {
			enabled = append(enabled, lang)
		}
	}
	return enabled
}

// Merge merges another config into this one (other takes precedence).
func (c *Config) Merge(other *Config) {
	if other == nil {
		return
	}

	// Merge languages config
	if len(other.Languages.Enabled) > 0 {
		c.Languages.Enabled = other.Languages.Enabled
	}
	if len(other.Languages.Disabled) > 0 {
		c.Languages.Disabled = append(c.Languages.Disabled, other.Languages.Disabled...)
	}

	// Merge Go config
	if other.Go.Enabled != nil {
		c.Go.Enabled = other.Go.Enabled
	}
	if other.Go.NamingConvention != "" {
		c.Go.NamingConvention = other.Go.NamingConvention
	}
	if other.Go.NamingConventionExternal != "" {
		c.Go.NamingConventionExternal = other.Go.NamingConventionExternal
	}

	// Merge Kotlin config
	if other.Kotlin.Enabled != nil {
		c.Kotlin.Enabled = other.Kotlin.Enabled
	}
	if other.Kotlin.ParserBackend != "" {
		c.Kotlin.ParserBackend = other.Kotlin.ParserBackend
	}
	if other.Kotlin.LibraryMacro != "" {
		c.Kotlin.LibraryMacro = other.Kotlin.LibraryMacro
	}
	if other.Kotlin.TestMacro != "" {
		c.Kotlin.TestMacro = other.Kotlin.TestMacro
	}
	if other.Kotlin.FQNScanning != nil {
		c.Kotlin.FQNScanning = other.Kotlin.FQNScanning
	}

	// Merge Python config
	if other.Python.Enabled != nil {
		c.Python.Enabled = other.Python.Enabled
	}
	if other.Python.StdlibModulesFile != "" {
		c.Python.StdlibModulesFile = other.Python.StdlibModulesFile
	}
	if other.Python.TestFramework != "" {
		c.Python.TestFramework = other.Python.TestFramework
	}
	if other.Python.LibraryMacro != "" {
		c.Python.LibraryMacro = other.Python.LibraryMacro
	}
	if other.Python.TestMacro != "" {
		c.Python.TestMacro = other.Python.TestMacro
	}
	if other.Python.BinaryMacro != "" {
		c.Python.BinaryMacro = other.Python.BinaryMacro
	}

	// Merge Java config
	if other.Java.Enabled != nil {
		c.Java.Enabled = other.Java.Enabled
	}
	if other.Java.LibraryMacro != "" {
		c.Java.LibraryMacro = other.Java.LibraryMacro
	}
	if other.Java.TestMacro != "" {
		c.Java.TestMacro = other.Java.TestMacro
	}

	// Merge Scala config
	if other.Scala.Enabled != nil {
		c.Scala.Enabled = other.Scala.Enabled
	}
	if other.Scala.LibraryMacro != "" {
		c.Scala.LibraryMacro = other.Scala.LibraryMacro
	}
	if other.Scala.TestMacro != "" {
		c.Scala.TestMacro = other.Scala.TestMacro
	}

	// Merge Groovy config
	if other.Groovy.Enabled != nil {
		c.Groovy.Enabled = other.Groovy.Enabled
	}
	if other.Groovy.LibraryMacro != "" {
		c.Groovy.LibraryMacro = other.Groovy.LibraryMacro
	}
	if other.Groovy.TestMacro != "" {
		c.Groovy.TestMacro = other.Groovy.TestMacro
	}
	if other.Groovy.SpockTestMacro != "" {
		c.Groovy.SpockTestMacro = other.Groovy.SpockTestMacro
	}

	// Merge Proto config
	if other.Proto.Enabled != nil {
		c.Proto.Enabled = other.Proto.Enabled
	}

	// Merge Rust config
	if other.Rust.Enabled != nil {
		c.Rust.Enabled = other.Rust.Enabled
	}

	// Merge CC config
	if other.CC.Enabled != nil {
		c.CC.Enabled = other.CC.Enabled
	}

	// Merge Bzl config
	if other.Bzl.Enabled != nil {
		c.Bzl.Enabled = other.Bzl.Enabled
	}
}
