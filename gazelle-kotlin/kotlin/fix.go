package kotlin

import (
	"github.com/albertocavalcante/bazelle/pkg/jvm"
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// Fix implements language.Language.
func (*kotlinLang) Fix(c *config.Config, f *rule.File) {
	jvm.DefaultFix(c, f)
}
