package groovy

import (
	"github.com/albertocavalcante/bazelle/pkg/jvm"
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/repo"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// Imports implements resolve.Resolver.
func (*groovyLang) Imports(c *config.Config, r *rule.Rule, f *rule.File) []resolve.ImportSpec {
	return jvm.DefaultImports(c, r, f)
}

// Embeds implements resolve.Resolver.
func (*groovyLang) Embeds(r *rule.Rule, from label.Label) []label.Label {
	return jvm.DefaultEmbeds(r, from)
}

// Resolve implements resolve.Resolver.
func (*groovyLang) Resolve(c *config.Config, ix *resolve.RuleIndex, rc *repo.RemoteCache, r *rule.Rule, imports interface{}, from label.Label) {
	jvm.DefaultResolve(c, ix, rc, r, imports, from)
}
