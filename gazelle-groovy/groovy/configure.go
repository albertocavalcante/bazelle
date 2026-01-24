package groovy

import (
	"flag"
	"log"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

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
	return []string{
		"groovy_enabled",
		"groovy_library_macro",
		"groovy_test_macro",
		"groovy_spock_test_macro",
		"groovy_spock_detection",
		"groovy_visibility",
		"groovy_load",
	}
}

// Configure implements config.Configurer.
func (*groovyLang) Configure(c *config.Config, rel string, f *rule.File) {
	gc := GetGroovyConfig(c)
	if gc == nil {
		gc = NewGroovyConfig()
		c.Exts[groovyName] = gc
	}

	newGc := *gc
	c.Exts[groovyName] = &newGc

	if f == nil {
		return
	}

	for _, d := range f.Directives {
		switch d.Key {
		case "groovy_enabled":
			newGc.Enabled = strings.ToLower(d.Value) == "true"
		case "groovy_library_macro":
			newGc.LibraryMacro = d.Value
		case "groovy_test_macro":
			newGc.TestMacro = d.Value
		case "groovy_spock_test_macro":
			newGc.SpockTestMacro = d.Value
		case "groovy_spock_detection":
			newGc.SpockDetection = strings.ToLower(d.Value) == "true"
		case "groovy_visibility":
			newGc.Visibility = d.Value
		case "groovy_load":
			if strings.Contains(d.Value, "..") {
				log.Printf("WARNING: groovy_load path contains '..' which may be unsafe: %s", d.Value)
			}
			newGc.LoadPath = d.Value
		}
	}
}
