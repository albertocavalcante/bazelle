package kotlin

import (
	"path/filepath"

	"github.com/albertocavalcante/bazelle/internal/log"
	"github.com/albertocavalcante/bazelle/pkg/jvm"
	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// GenerateRules implements language.Language.
func (k *kotlinLang) GenerateRules(args language.GenerateArgs) language.GenerateResult {
	kc := GetKotlinConfig(args.Config)

	if !kc.Enabled {
		return language.GenerateResult{}
	}

	// Find Kotlin source files using jvm package
	mainFiles := jvm.FindMainSources(args.Dir, jvm.Kotlin)
	testFiles := jvm.FindTestSources(args.Dir, jvm.Kotlin)

	if len(mainFiles) == 0 && len(testFiles) == 0 {
		return language.GenerateResult{}
	}

	var rules []*rule.Rule
	var imports []interface{}

	// Generate library rule for main sources
	if len(mainFiles) > 0 {
		libRule := k.generateLibraryRule(args, kc, mainFiles)
		if libRule != nil {
			rules = append(rules, libRule)
			imports = append(imports, nil)
		}
	}

	// Generate test rule for test sources
	if len(testFiles) > 0 {
		testRule := k.generateTestRule(args, kc, testFiles, len(mainFiles) > 0)
		if testRule != nil {
			rules = append(rules, testRule)
			imports = append(imports, nil)
		}
	}

	return language.GenerateResult{
		Gen:     rules,
		Imports: imports,
	}
}

// generateLibraryRule creates a kt_jvm_library (or custom macro) rule.
func (k *kotlinLang) generateLibraryRule(args language.GenerateArgs, kc *KotlinConfig, files []string) *rule.Rule {
	// Derive target name from directory name using jvm package
	name := jvm.DeriveTargetName(args.Dir, args.Config.RepoRoot)

	r := rule.NewRule(kc.LibraryMacro, name)
	r.SetAttr("srcs", rule.GlobValue{Patterns: []string{
		"src/main/kotlin/**/*.kt",
		"src/main/kotlin/**/*.kts",
	}})
	r.SetAttr("visibility", []string{kc.Visibility})

	// Parse files to get package info
	fullPaths := make([]string, len(files))
	for i, f := range files {
		fullPaths[i] = filepath.Join(args.Dir, f)
	}

	results, err := k.parser.ParseFiles(fullPaths)
	if err != nil {
		log.Warn("failed to parse kotlin files",
			"target", name, "error", err)
	}
	if len(results) > 0 {
		r.SetPrivateAttr("packages", GetPackages(results))
	}

	return r
}

// generateTestRule creates a kt_jvm_test (or custom macro) rule.
func (k *kotlinLang) generateTestRule(args language.GenerateArgs, kc *KotlinConfig, files []string, hasMain bool) *rule.Rule {
	// Derive target name from directory name using jvm package
	baseName := jvm.DeriveTargetName(args.Dir, args.Config.RepoRoot)
	name := jvm.DeriveTestTargetName(args.Dir, args.Config.RepoRoot)

	r := rule.NewRule(kc.TestMacro, name)
	r.SetAttr("srcs", rule.GlobValue{Patterns: []string{
		"src/test/kotlin/**/*.kt",
		"src/test/kotlin/**/*.kts",
	}})

	// Parse files to get test packages
	fullPaths := make([]string, len(files))
	for i, f := range files {
		fullPaths[i] = filepath.Join(args.Dir, f)
	}

	results, err := k.parser.ParseFiles(fullPaths)
	if err != nil {
		log.Warn("failed to parse kotlin test files",
			"target", name, "error", err)
	}
	if len(results) > 0 {
		packages := GetPackages(results)
		if len(packages) > 0 {
			r.SetAttr("test_packages", packages)
		}
	}

	// Add associate to the library
	if hasMain {
		r.SetAttr("associates", []string{":" + baseName})
		r.SetAttr("deps", []string{":" + baseName})
	}

	return r
}

