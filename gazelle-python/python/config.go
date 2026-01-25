package python

import (
	"flag"
	"log"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// PythonConfig holds configuration for the Python extension.
type PythonConfig struct {
	// Enabled specifies whether the Python extension is enabled.
	Enabled bool

	// LibraryMacro is the macro to use for py_library rules.
	LibraryMacro string

	// TestMacro is the macro to use for py_test rules.
	TestMacro string

	// BinaryMacro is the macro to use for py_binary rules.
	BinaryMacro string

	// Visibility is the default visibility for generated rules.
	Visibility string

	// LoadPath is the path for loading custom macros.
	LoadPath string

	// TestFramework is the test framework to use ("pytest", "unittest").
	TestFramework string

	// StdlibModulesFile is an optional path to a custom stdlib modules list.
	StdlibModulesFile string
}

// Clone creates a copy of the configuration.
func (c *PythonConfig) Clone() *PythonConfig {
	clone := *c
	return &clone
}

// NewPythonConfig creates a new PythonConfig with default values.
func NewPythonConfig() *PythonConfig {
	return &PythonConfig{
		Enabled:       false,
		LibraryMacro:  "py_library",
		TestMacro:     "py_test",
		BinaryMacro:   "py_binary",
		Visibility:    "//visibility:public",
		TestFramework: "pytest",
	}
}

// GetPythonConfig extracts PythonConfig from the Gazelle config.
func GetPythonConfig(c *config.Config) *PythonConfig {
	pc, ok := c.Exts[pythonName].(*PythonConfig)
	if !ok {
		return NewPythonConfig()
	}
	return pc
}

// RegisterFlags implements config.Configurer.
func (*pythonLang) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	pc := NewPythonConfig()
	c.Exts[pythonName] = pc
}

// CheckFlags implements config.Configurer.
func (*pythonLang) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	return nil
}

// KnownDirectives implements config.Configurer.
func (*pythonLang) KnownDirectives() []string {
	return []string{
		"python_enabled",
		"python_library_macro",
		"python_test_macro",
		"python_binary_macro",
		"python_visibility",
		"python_load",
		"python_test_framework",
		"python_stdlib_modules_file",
	}
}

// Configure implements config.Configurer.
func (*pythonLang) Configure(c *config.Config, rel string, f *rule.File) {
	pc := GetPythonConfig(c)
	if pc == nil {
		pc = NewPythonConfig()
		c.Exts[pythonName] = pc
	}

	// Create a new config for this directory (inheriting from parent)
	newPc := pc.Clone()
	c.Exts[pythonName] = newPc

	if f == nil {
		return
	}

	// Process directives
	for _, d := range f.Directives {
		switch d.Key {
		case "python_enabled":
			newPc.Enabled = strings.ToLower(d.Value) == "true"
		case "python_library_macro":
			newPc.LibraryMacro = d.Value
		case "python_test_macro":
			newPc.TestMacro = d.Value
		case "python_binary_macro":
			newPc.BinaryMacro = d.Value
		case "python_visibility":
			newPc.Visibility = d.Value
		case "python_load":
			newPc.LoadPath = d.Value
		case "python_test_framework":
			switch strings.ToLower(d.Value) {
			case "pytest", "unittest":
				newPc.TestFramework = strings.ToLower(d.Value)
			default:
				log.Printf("WARNING: unknown python_test_framework %q, using pytest", d.Value)
				newPc.TestFramework = "pytest"
			}
		case "python_stdlib_modules_file":
			newPc.StdlibModulesFile = d.Value
		}
	}
}
