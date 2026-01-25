package kotlin

import (
	"flag"
	"strings"

	"github.com/albertocavalcante/bazelle/internal/log"
	"github.com/albertocavalcante/bazelle/pkg/jvm"
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// KotlinConfig holds configuration for the Kotlin extension.
type KotlinConfig struct {
	// Embed BaseConfig for common JVM language fields.
	jvm.BaseConfig

	// ParserBackend specifies which parsing strategy to use.
	// Options: "heuristic" (default), "treesitter", "hybrid"
	ParserBackend ParserBackendType

	// EnableFQNScanning enables detection of fully-qualified names in code body.
	EnableFQNScanning bool
}

// Clone implements jvm.Config.
func (c *KotlinConfig) Clone() jvm.Config {
	clone := *c
	return &clone
}

// NewKotlinConfig creates a new KotlinConfig with default values.
func NewKotlinConfig() *KotlinConfig {
	return &KotlinConfig{
		BaseConfig:        jvm.NewBaseConfig(jvm.Kotlin),
		ParserBackend:     BackendHeuristic,
		EnableFQNScanning: true,
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
	directives := jvm.CommonDirectiveNames(jvm.Kotlin)
	// Add Kotlin-specific directives
	directives = append(directives,
		"kotlin_parser_backend",
		"kotlin_fqn_scanning",
	)
	return directives
}

// Configure implements config.Configurer.
func (*kotlinLang) Configure(c *config.Config, rel string, f *rule.File) {
	kc := GetKotlinConfig(c)
	if kc == nil {
		kc = NewKotlinConfig()
		c.Exts[kotlinName] = kc
	}

	// Create a new config for this directory (inheriting from parent)
	newKc := kc.Clone().(*KotlinConfig)
	c.Exts[kotlinName] = newKc

	if f == nil {
		return
	}

	// Build handlers: common JVM directives + Kotlin-specific
	handlers := jvm.CommonDirectives(jvm.Kotlin)
	handlers["kotlin_parser_backend"] = func(cfg jvm.Config, value string) {
		kc := cfg.(*KotlinConfig)
		switch strings.ToLower(value) {
		case "heuristic":
			kc.ParserBackend = BackendHeuristic
		case "treesitter":
			kc.ParserBackend = BackendTreeSitter
		case "hybrid":
			kc.ParserBackend = BackendHybrid
		default:
			log.Warn("unknown kotlin_parser_backend, using heuristic",
				"value", value, "language", "kotlin")
			kc.ParserBackend = BackendHeuristic
		}
	}
	handlers["kotlin_fqn_scanning"] = func(cfg jvm.Config, value string) {
		cfg.(*KotlinConfig).EnableFQNScanning = strings.ToLower(value) == "true"
	}

	jvm.ProcessDirectives(f, newKc, handlers)
}
