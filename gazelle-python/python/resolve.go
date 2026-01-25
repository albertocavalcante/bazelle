package python

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/repo"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// Imports implements resolve.Resolver.
func (*pythonLang) Imports(c *config.Config, r *rule.Rule, f *rule.File) []resolve.ImportSpec {
	// Return import specs for this rule
	// The imports are stored as a private attribute during generation
	if imports := r.PrivateAttr("python_imports"); imports != nil {
		if importList, ok := imports.([]string); ok {
			specs := make([]resolve.ImportSpec, 0, len(importList))
			for _, imp := range importList {
				specs = append(specs, resolve.ImportSpec{
					Lang: pythonName,
					Imp:  imp,
				})
			}
			return specs
		}
	}
	return nil
}

// Embeds implements resolve.Resolver.
func (*pythonLang) Embeds(r *rule.Rule, from label.Label) []label.Label {
	// Python doesn't have an embed concept like Go
	return nil
}

// Resolve implements resolve.Resolver.
func (*pythonLang) Resolve(c *config.Config, ix *resolve.RuleIndex, rc *repo.RemoteCache, r *rule.Rule, imports interface{}, from label.Label) {
	pc := GetPythonConfig(c)
	if !pc.Enabled {
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
		if IsStdlib(imp) {
			continue
		}

		// Try to resolve using the rule index
		spec := resolve.ImportSpec{
			Lang: pythonName,
			Imp:  imp,
		}
		if matches := ix.FindRulesByImport(spec, pythonName); len(matches) > 0 {
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
		// that would be handled by pip_install or similar
	}

	if len(deps) > 0 {
		r.SetAttr("deps", deps)
	}
}
