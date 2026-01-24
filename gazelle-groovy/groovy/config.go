package groovy

import "github.com/bazelbuild/bazel-gazelle/config"

// GroovyConfig holds configuration for the Groovy extension.
type GroovyConfig struct {
	Enabled        bool
	LibraryMacro   string
	TestMacro      string
	SpockTestMacro string
	Visibility     string
	LoadPath       string
	SpockDetection bool
}

// NewGroovyConfig creates a new GroovyConfig with default values.
func NewGroovyConfig() *GroovyConfig {
	return &GroovyConfig{
		Enabled:        false,
		LibraryMacro:   "groovy_library",
		TestMacro:      "groovy_test",
		SpockTestMacro: "spock_test",
		Visibility:     "//visibility:public",
		LoadPath:       "",
		SpockDetection: true,
	}
}

// GetGroovyConfig extracts GroovyConfig from the Gazelle config.
func GetGroovyConfig(c *config.Config) *GroovyConfig {
	gc, ok := c.Exts[groovyName].(*GroovyConfig)
	if !ok || gc == nil {
		return NewGroovyConfig()
	}
	return gc
}
