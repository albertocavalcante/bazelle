package jvm

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// DefaultFix performs default fixes on a BUILD file.
// For MVP implementations, this is a no-op.
func DefaultFix(c *config.Config, f *rule.File) {
	// No-op for MVP
}

// FixGlobPatterns updates glob patterns to match the current standards.
func FixGlobPatterns(r *rule.Rule, lang Language) {
	// Future: Update old-style globs to new patterns
}

// FixDeprecatedAttrs replaces deprecated attributes with their modern equivalents.
func FixDeprecatedAttrs(r *rule.Rule) {
	// Future: Handle attribute migrations
}

// MigrateMacro migrates a rule from one macro to another.
func MigrateMacro(r *rule.Rule, fromMacro, toMacro string) {
	// Future: Handle macro migrations
}
