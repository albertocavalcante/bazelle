package groovy

import "github.com/bazelbuild/bazel-gazelle/rule"

// Kinds implements language.Language.
func (*groovyLang) Kinds() map[string]rule.KindInfo {
	return map[string]rule.KindInfo{
		"groovy_library": {
			MatchAny:        false,
			NonEmptyAttrs:   map[string]bool{"srcs": true},
			SubstituteAttrs: map[string]bool{},
			MergeableAttrs:  map[string]bool{"srcs": true, "deps": true},
			ResolveAttrs:    map[string]bool{"deps": true},
		},
		"groovy_test": {
			MatchAny:        false,
			NonEmptyAttrs:   map[string]bool{"srcs": true},
			SubstituteAttrs: map[string]bool{},
			MergeableAttrs:  map[string]bool{"srcs": true, "deps": true},
			ResolveAttrs:    map[string]bool{"deps": true},
		},
		"spock_test": {
			MatchAny:        false,
			NonEmptyAttrs:   map[string]bool{"specs": true},
			SubstituteAttrs: map[string]bool{},
			MergeableAttrs:  map[string]bool{"specs": true, "deps": true},
			ResolveAttrs:    map[string]bool{"deps": true},
		},
	}
}

// Loads implements language.Language.
func (*groovyLang) Loads() []rule.LoadInfo {
	return []rule.LoadInfo{
		{
			Name:    "@rules_groovy//groovy:groovy.bzl",
			Symbols: []string{"groovy_library", "groovy_test", "spock_test"},
		},
	}
}
