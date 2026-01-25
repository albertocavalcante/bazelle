package python

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// Fix implements language.Language.
// Currently a no-op - future versions may fix up existing Python rules.
func (*pythonLang) Fix(c *config.Config, f *rule.File) {
	// No-op for now
}
