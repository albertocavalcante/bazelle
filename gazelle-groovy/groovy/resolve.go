package groovy

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/repo"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// Imports implements resolve.Resolver.
// Returns import specs for this rule based on parsed Groovy imports.
func (*groovyLang) Imports(c *config.Config, r *rule.Rule, f *rule.File) []resolve.ImportSpec {
	// Get imports from private attribute set during generation
	if imports := r.PrivateAttr("groovy_imports"); imports != nil {
		if importList, ok := imports.([]string); ok {
			specs := make([]resolve.ImportSpec, 0, len(importList))
			for _, imp := range importList {
				specs = append(specs, resolve.ImportSpec{
					Lang: groovyName,
					Imp:  imp,
				})
			}
			return specs
		}
	}
	return nil
}

// Embeds implements resolve.Resolver.
func (*groovyLang) Embeds(r *rule.Rule, from label.Label) []label.Label {
	// Groovy doesn't have an embed concept
	return nil
}

// Resolve implements resolve.Resolver.
// Resolves Groovy imports to Bazel dependencies.
func (*groovyLang) Resolve(c *config.Config, ix *resolve.RuleIndex, rc *repo.RemoteCache, r *rule.Rule, imports interface{}, from label.Label) {
	gc := GetGroovyConfig(c)
	if !gc.Enabled {
		return
	}

	// Get the imports for this rule
	importList, ok := imports.([]string)
	if !ok || len(importList) == 0 {
		return
	}

	// Resolve each import to a dependency
	var deps []string
	for _, imp := range importList {
		// Skip stdlib imports
		if IsGroovyStdlib(imp) {
			continue
		}

		// Try to resolve using the rule index
		spec := resolve.ImportSpec{
			Lang: groovyName,
			Imp:  imp,
		}
		if matches := ix.FindRulesByImport(spec, groovyName); len(matches) > 0 {
			// Use the first match
			l := matches[0].Label
			if l.Repo == "" && l.Pkg == from.Pkg {
				// Same package, use relative label
				deps = append(deps, ":"+l.Name)
			} else {
				deps = append(deps, l.String())
			}
		}
		// If not found in index, the import might be an external dependency
	}

	if len(deps) > 0 {
		// Merge with existing deps
		existingDeps := r.AttrStrings("deps")
		seen := make(map[string]bool)
		for _, d := range existingDeps {
			seen[d] = true
		}
		for _, d := range deps {
			if !seen[d] {
				existingDeps = append(existingDeps, d)
				seen[d] = true
			}
		}
		r.SetAttr("deps", existingDeps)
	}
}
