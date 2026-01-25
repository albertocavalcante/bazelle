package jvm

import (
	"log"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/rule"
)

// DirectiveHandler is a function that processes a directive value.
type DirectiveHandler func(cfg Config, value string)

// CommonDirectives returns the standard directive handlers for a JVM language.
// The handlers are keyed by the full directive name (e.g., "kotlin_enabled").
func CommonDirectives(lang Language) map[string]DirectiveHandler {
	prefix := lang.DirectivePrefix()
	return map[string]DirectiveHandler{
		prefix + "_enabled": func(cfg Config, value string) {
			cfg.SetEnabled(strings.ToLower(value) == "true")
		},
		prefix + "_library_macro": func(cfg Config, value string) {
			cfg.SetLibraryMacro(value)
		},
		prefix + "_test_macro": func(cfg Config, value string) {
			cfg.SetTestMacro(value)
		},
		prefix + "_visibility": func(cfg Config, value string) {
			cfg.SetVisibility(value)
		},
		prefix + "_load": func(cfg Config, value string) {
			WarnIfPathTraversal(prefix+"_load", value)
			cfg.SetLoadPath(value)
		},
	}
}

// CommonDirectiveNames returns the list of common directive names for a language.
func CommonDirectiveNames(lang Language) []string {
	prefix := lang.DirectivePrefix()
	return []string{
		prefix + "_enabled",
		prefix + "_library_macro",
		prefix + "_test_macro",
		prefix + "_visibility",
		prefix + "_load",
	}
}

// ProcessDirectives processes all directives in a rule.File using the provided handlers.
func ProcessDirectives(f *rule.File, cfg Config, handlers map[string]DirectiveHandler) {
	if f == nil {
		return
	}

	for _, d := range f.Directives {
		if handler, ok := handlers[d.Key]; ok {
			handler(cfg, d.Value)
		}
	}
}

// WarnIfPathTraversal logs a warning if the path contains ".." which may be unsafe.
func WarnIfPathTraversal(directive, value string) {
	if strings.Contains(value, "..") {
		log.Printf("WARNING: %s path contains '..' which may be unsafe: %s", directive, value)
	}
}

// MergeHandlers combines multiple handler maps into one.
// Later maps override earlier ones for the same key.
func MergeHandlers(handlers ...map[string]DirectiveHandler) map[string]DirectiveHandler {
	result := make(map[string]DirectiveHandler)
	for _, h := range handlers {
		for k, v := range h {
			result[k] = v
		}
	}
	return result
}
