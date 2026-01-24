package groovy

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// Fix implements language.Language.
func (*groovyLang) Fix(c *config.Config, f *rule.File) {
}
