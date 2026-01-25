package python

import (
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// Kinds implements language.Language.
func (*pythonLang) Kinds() map[string]rule.KindInfo {
	return map[string]rule.KindInfo{
		"py_library": {
			MatchAny: false,
			NonEmptyAttrs: map[string]bool{
				"srcs": true,
			},
			SubstituteAttrs: map[string]bool{},
			MergeableAttrs: map[string]bool{
				"srcs": true,
				"deps": true,
			},
			ResolveAttrs: map[string]bool{
				"deps": true,
			},
		},
		"py_binary": {
			MatchAny: false,
			NonEmptyAttrs: map[string]bool{
				"srcs": true,
			},
			SubstituteAttrs: map[string]bool{},
			MergeableAttrs: map[string]bool{
				"srcs": true,
				"deps": true,
			},
			ResolveAttrs: map[string]bool{
				"deps": true,
			},
		},
		"py_test": {
			MatchAny: false,
			NonEmptyAttrs: map[string]bool{
				"srcs": true,
			},
			SubstituteAttrs: map[string]bool{},
			MergeableAttrs: map[string]bool{
				"srcs": true,
				"deps": true,
			},
			ResolveAttrs: map[string]bool{
				"deps": true,
			},
		},
	}
}

// Loads implements language.Language.
func (*pythonLang) Loads() []rule.LoadInfo {
	return []rule.LoadInfo{
		{
			Name:    "@rules_python//python:defs.bzl",
			Symbols: []string{"py_library", "py_binary", "py_test"},
		},
	}
}
