package groovy

import (
	"flag"
	"strings"

	"github.com/albertocavalcante/bazelle/pkg/jvm"
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// GroovyConfig holds configuration for the Groovy extension.
type GroovyConfig struct {
	// Embed BaseConfig for common JVM language fields.
	jvm.BaseConfig

	// SpockTestMacro is the rule kind to use for Spock tests.
	SpockTestMacro string

	// SpockDetection enables automatic detection of Spock specification files.
	SpockDetection bool
}

// Clone implements jvm.Config.
func (c *GroovyConfig) Clone() jvm.Config {
	clone := *c
	return &clone
}

// NewGroovyConfig creates a new GroovyConfig with default values.
func NewGroovyConfig() *GroovyConfig {
	return &GroovyConfig{
		BaseConfig:     jvm.NewBaseConfig(jvm.Groovy),
		SpockTestMacro: "spock_test",
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

// RegisterFlags implements config.Configurer.
func (*groovyLang) RegisterFlags(fs *flag.FlagSet, cmd string, c *config.Config) {
	gc := NewGroovyConfig()
	c.Exts[groovyName] = gc
}

// CheckFlags implements config.Configurer.
func (*groovyLang) CheckFlags(fs *flag.FlagSet, c *config.Config) error {
	return nil
}

// KnownDirectives implements config.Configurer.
func (*groovyLang) KnownDirectives() []string {
	directives := jvm.CommonDirectiveNames(jvm.Groovy)
	// Add Groovy-specific directives
	directives = append(directives,
		"groovy_spock_test_macro",
		"groovy_spock_detection",
	)
	return directives
}

// Configure implements config.Configurer.
func (*groovyLang) Configure(c *config.Config, rel string, f *rule.File) {
	gc := GetGroovyConfig(c)
	if gc == nil {
		gc = NewGroovyConfig()
		c.Exts[groovyName] = gc
	}

	// Create a new config for this directory (inheriting from parent)
	newGc := gc.Clone().(*GroovyConfig)
	c.Exts[groovyName] = newGc

	if f == nil {
		return
	}

	// Build handlers: common JVM directives + Groovy-specific
	handlers := jvm.CommonDirectives(jvm.Groovy)
	handlers["groovy_spock_test_macro"] = func(cfg jvm.Config, value string) {
		cfg.(*GroovyConfig).SpockTestMacro = value
	}
	handlers["groovy_spock_detection"] = func(cfg jvm.Config, value string) {
		cfg.(*GroovyConfig).SpockDetection = strings.ToLower(value) == "true"
	}

	jvm.ProcessDirectives(f, newGc, handlers)
}
