package jvm

import "github.com/bazelbuild/bazel-gazelle/rule"

// RuleOptions contains options for creating a new rule.
type RuleOptions struct {
	// Kind is the rule kind (e.g., "kt_jvm_library").
	Kind string

	// Name is the target name.
	Name string

	// SrcsPatterns are glob patterns for source files.
	SrcsPatterns []string

	// SrcsExcludes are glob patterns to exclude from sources.
	SrcsExcludes []string

	// Visibility is the visibility attribute value.
	Visibility string

	// Deps are explicit dependencies.
	Deps []string

	// TestOnly indicates this is a test-only target.
	TestOnly bool
}

// NewRule creates a new rule with the given options.
func NewRule(opts RuleOptions) *rule.Rule {
	r := rule.NewRule(opts.Kind, opts.Name)

	// Set srcs attribute
	if len(opts.SrcsPatterns) > 0 {
		glob := rule.GlobValue{
			Patterns: opts.SrcsPatterns,
		}
		if len(opts.SrcsExcludes) > 0 {
			glob.Excludes = opts.SrcsExcludes
		}
		r.SetAttr("srcs", glob)
	}

	// Set visibility
	if opts.Visibility != "" {
		r.SetAttr("visibility", []string{opts.Visibility})
	}

	// Set deps
	if len(opts.Deps) > 0 {
		r.SetAttr("deps", opts.Deps)
	}

	// Set testonly
	if opts.TestOnly {
		r.SetAttr("testonly", true)
	}

	return r
}

// NewLibraryRule creates a library rule for the given language.
func NewLibraryRule(lang Language, dir, repoRoot, macro, visibility string) *rule.Rule {
	name := DeriveTargetName(dir, repoRoot)
	return NewRule(RuleOptions{
		Kind:         macro,
		Name:         name,
		SrcsPatterns: lang.GlobPatterns(lang.MainSourceDir()),
		Visibility:   visibility,
	})
}

// NewTestRule creates a test rule for the given language.
func NewTestRule(lang Language, dir, repoRoot, macro string, hasMain bool) *rule.Rule {
	name := DeriveTestTargetName(dir, repoRoot)
	opts := RuleOptions{
		Kind:         macro,
		Name:         name,
		SrcsPatterns: lang.GlobPatterns(lang.TestSourceDir()),
	}

	if hasMain {
		opts.Deps = []string{DeriveLibraryLabel(dir, repoRoot)}
	}

	return NewRule(opts)
}

// AddDependency adds a dependency to an existing rule.
func AddDependency(r *rule.Rule, dep string) {
	existing := r.AttrStrings("deps")
	for _, d := range existing {
		if d == dep {
			return // already present
		}
	}
	r.SetAttr("deps", append(existing, dep))
}

// SetAssociates sets the associates attribute for a Kotlin test rule.
// This is used for test rules that need access to internal declarations.
func SetAssociates(r *rule.Rule, associates []string) {
	if len(associates) > 0 {
		r.SetAttr("associates", associates)
	}
}

// SetSpecs sets the specs attribute for a Spock test rule.
func SetSpecs(r *rule.Rule, patterns []string, excludes []string) {
	glob := rule.GlobValue{Patterns: patterns}
	if len(excludes) > 0 {
		glob.Excludes = excludes
	}
	r.SetAttr("specs", glob)
}

// SetGroovySrcs sets the groovy_srcs attribute for non-spec files in a Spock test.
func SetGroovySrcs(r *rule.Rule, patterns []string, excludes []string) {
	glob := rule.GlobValue{Patterns: patterns}
	if len(excludes) > 0 {
		glob.Excludes = excludes
	}
	r.SetAttr("groovy_srcs", glob)
}
