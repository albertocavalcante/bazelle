package kotlin

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/repo"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// Imports implements resolve.Resolver.
func (*kotlinLang) Imports(c *config.Config, r *rule.Rule, f *rule.File) []resolve.ImportSpec {
	// For MVP, we don't extract imports for resolution
	// This will be implemented in Phase 2
	return nil
}

// Embeds implements resolve.Resolver.
func (*kotlinLang) Embeds(r *rule.Rule, from label.Label) []label.Label {
	return nil
}

// Resolve implements resolve.Resolver.
func (*kotlinLang) Resolve(c *config.Config, ix *resolve.RuleIndex, rc *repo.RemoteCache, r *rule.Rule, imports interface{}, from label.Label) {
	// For MVP, we don't resolve dependencies automatically
	// Users add deps manually after generation
	// This will be implemented in Phase 2
}
