package jvm

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/repo"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// DefaultImports returns the default imports for a rule.
// For MVP implementations, this returns nil as import extraction
// will be implemented in a later phase.
func DefaultImports(c *config.Config, r *rule.Rule, f *rule.File) []resolve.ImportSpec {
	return nil
}

// DefaultEmbeds returns the default embeds for a rule.
// For most JVM languages, this returns nil.
func DefaultEmbeds(r *rule.Rule, from label.Label) []label.Label {
	return nil
}

// DefaultResolve performs default dependency resolution for a rule.
// For MVP implementations, this is a no-op as automatic resolution
// will be implemented in a later phase.
func DefaultResolve(c *config.Config, ix *resolve.RuleIndex, rc *repo.RemoteCache, r *rule.Rule, imports interface{}, from label.Label) {
	// For MVP, we don't resolve dependencies automatically.
	// Users add deps manually after generation.
	// This will be implemented in a later phase.
}

// ImportSpec creates an ImportSpec for a JVM import.
func ImportSpec(lang Language, imp string) resolve.ImportSpec {
	return resolve.ImportSpec{
		Lang: string(lang),
		Imp:  imp,
	}
}

// CrossResolve provides cross-language resolution hints.
// This allows JVM languages to depend on each other.
func CrossResolve(c *config.Config, ix *resolve.RuleIndex, imp resolve.ImportSpec, lang Language) []resolve.FindResult {
	// Check if the import can be resolved within our language
	results := ix.FindRulesByImportWithConfig(c, imp, string(lang))
	if len(results) > 0 {
		return results
	}

	// Try cross-language resolution for JVM languages
	for _, otherLang := range AllLanguages() {
		if otherLang != lang {
			crossImp := resolve.ImportSpec{
				Lang: string(otherLang),
				Imp:  imp.Imp,
			}
			results = ix.FindRulesByImportWithConfig(c, crossImp, string(otherLang))
			if len(results) > 0 {
				return results
			}
		}
	}

	return nil
}
