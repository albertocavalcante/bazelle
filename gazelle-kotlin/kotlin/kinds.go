package kotlin

import (
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// Kinds implements language.Language.
func (*kotlinLang) Kinds() map[string]rule.KindInfo {
	return map[string]rule.KindInfo{
		// Native rules_kotlin rules
		"kt_jvm_library": {
			MatchAny:       false,
			NonEmptyAttrs:  map[string]bool{"srcs": true},
			SubstituteAttrs: map[string]bool{},
			MergeableAttrs: map[string]bool{"srcs": true, "deps": true},
			ResolveAttrs:   map[string]bool{"deps": true},
		},
		"kt_jvm_test": {
			MatchAny:       false,
			NonEmptyAttrs:  map[string]bool{"srcs": true},
			SubstituteAttrs: map[string]bool{},
			MergeableAttrs: map[string]bool{"srcs": true, "deps": true},
			ResolveAttrs:   map[string]bool{"deps": true},
		},
		"kt_jvm_binary": {
			MatchAny:       false,
			NonEmptyAttrs:  map[string]bool{"srcs": true},
			SubstituteAttrs: map[string]bool{},
			MergeableAttrs: map[string]bool{"srcs": true, "deps": true},
			ResolveAttrs:   map[string]bool{"deps": true},
		},
		// Custom macros (commonly used)
		"kt_library": {
			MatchAny:       false,
			NonEmptyAttrs:  map[string]bool{"srcs": true},
			SubstituteAttrs: map[string]bool{},
			MergeableAttrs: map[string]bool{"srcs": true, "deps": true},
			ResolveAttrs:   map[string]bool{"deps": true},
		},
		"kt_test": {
			MatchAny:       false,
			NonEmptyAttrs:  map[string]bool{"srcs": true},
			SubstituteAttrs: map[string]bool{},
			MergeableAttrs: map[string]bool{"srcs": true, "deps": true},
			ResolveAttrs:   map[string]bool{"deps": true},
		},
	}
}

// Loads implements language.Language.
func (k *kotlinLang) Loads() []rule.LoadInfo {
	return []rule.LoadInfo{
		{
			Name:    "@rules_kotlin//kotlin:jvm.bzl",
			Symbols: []string{"kt_jvm_library", "kt_jvm_test", "kt_jvm_binary"},
		},
	}
}
