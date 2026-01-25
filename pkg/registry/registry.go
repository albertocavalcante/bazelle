// Package registry provides a language factory registry for dynamic language loading.
// It allows Bazelle to load only the language extensions that are enabled in configuration.
package registry

import (
	"github.com/albertocavalcante/bazelle/gazelle-groovy/groovy"
	"github.com/albertocavalcante/bazelle/gazelle-kotlin/kotlin"
	"github.com/albertocavalcante/bazelle/gazelle-python/python"
	"github.com/albertocavalcante/bazelle/pkg/config"
	"github.com/bazelbuild/bazel-gazelle/language"
	goLang "github.com/bazelbuild/bazel-gazelle/language/go"
	protoLang "github.com/bazelbuild/bazel-gazelle/language/proto"
	ccLang "github.com/EngFlow/gazelle_cc/language/cc"
	// TODO: Re-enable when gazelle_rust proto dependency issue is fixed
	// rustLang "github.com/calsign/gazelle_rust/rust_language"
)

// LanguageFactory is a function that creates a new language extension.
type LanguageFactory func() language.Language

// factories maps language names to their factory functions.
// Languages are listed in order: proto first, then others.
var factories = map[string]LanguageFactory{
	"proto":  protoLang.NewLanguage,
	"go":     goLang.NewLanguage,
	"kotlin": kotlin.NewLanguage,
	"groovy": groovy.NewLanguage,
	"python": python.NewLanguage,
	"cc":     ccLang.NewLanguage,
	// TODO: Re-enable when dependencies are available
	// "rust":   rustLang.NewLanguage,
	// "java":   javaLang.NewLanguage,
	// "scala":  scalaLang.NewLanguage,
	// "bzl":    bzlLang.NewLanguage,
}

// languageOrder defines the order in which languages should be loaded.
// Proto should come first to establish .proto dependencies before other languages.
var languageOrder = []string{
	"proto",
	"go",
	"bzl",
	"java",
	"scala",
	"python",
	"cc",
	"kotlin",
	"groovy",
	"rust",
}

// LoadLanguages loads language extensions based on the configuration.
// Languages are returned in a consistent order (proto first, etc.).
func LoadLanguages(cfg *config.Config) []language.Language {
	enabled := cfg.GetEnabledLanguages()

	// Build a set of enabled languages for quick lookup
	enabledSet := make(map[string]bool, len(enabled))
	for _, lang := range enabled {
		enabledSet[lang] = true
	}

	// Load languages in the correct order
	var languages []language.Language
	for _, name := range languageOrder {
		if !enabledSet[name] {
			continue
		}
		factory, ok := factories[name]
		if !ok {
			// Language not available (e.g., java, scala, bzl)
			continue
		}
		languages = append(languages, factory())
	}

	return languages
}

// LoadLanguagesByName loads specific language extensions by name.
func LoadLanguagesByName(names []string) []language.Language {
	var languages []language.Language
	for _, name := range names {
		factory, ok := factories[name]
		if !ok {
			continue
		}
		languages = append(languages, factory())
	}
	return languages
}

// AvailableLanguages returns the list of available language names.
func AvailableLanguages() []string {
	var names []string
	for name := range factories {
		names = append(names, name)
	}
	return names
}

// IsLanguageAvailable checks if a language factory is registered.
func IsLanguageAvailable(name string) bool {
	_, ok := factories[name]
	return ok
}

// RegisterLanguage registers a language factory.
// This allows external packages to add new languages.
func RegisterLanguage(name string, factory LanguageFactory) {
	factories[name] = factory
}
