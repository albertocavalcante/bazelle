package kotlin

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// Fix implements language.Language.
func (*kotlinLang) Fix(c *config.Config, f *rule.File) {
	// For MVP, we don't fix existing BUILD files
	// This can be used in the future to:
	// - Update glob patterns
	// - Fix deprecated attributes
	// - Migrate from one macro to another
}
