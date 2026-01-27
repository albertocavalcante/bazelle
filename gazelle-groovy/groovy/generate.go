package groovy

import (
	"path/filepath"
	"strings"

	"github.com/albertocavalcante/bazelle/internal/log"
	"github.com/albertocavalcante/bazelle/pkg/jvm"
	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// GenerateRules implements language.Language.
func (g *groovyLang) GenerateRules(args language.GenerateArgs) language.GenerateResult {
	gc := GetGroovyConfig(args.Config)
	if !gc.Enabled {
		return language.GenerateResult{}
	}

	// Find Groovy source files using jvm package
	mainFiles := jvm.FindMainSources(args.Dir, jvm.Groovy)
	testFiles := jvm.FindTestSources(args.Dir, jvm.Groovy)

	if len(mainFiles) == 0 && len(testFiles) == 0 {
		return language.GenerateResult{}
	}

	var rules []*rule.Rule
	var imports []interface{}

	if len(mainFiles) > 0 {
		libRule, libImports := g.generateLibraryRule(args, gc, mainFiles)
		if libRule != nil {
			rules = append(rules, libRule)
			imports = append(imports, libImports)
		}
	}

	if len(testFiles) > 0 {
		testRule, testImports := g.generateTestRule(args, gc, testFiles, len(mainFiles) > 0)
		if testRule != nil {
			rules = append(rules, testRule)
			imports = append(imports, testImports)
		}
	}

	return language.GenerateResult{
		Gen:     rules,
		Imports: imports,
	}
}


func (g *groovyLang) generateLibraryRule(args language.GenerateArgs, gc *GroovyConfig, files []string) (*rule.Rule, []string) {
	// Derive target name from directory name using jvm package
	name := jvm.DeriveTargetName(args.Dir, args.Config.RepoRoot)

	r := rule.NewRule(gc.LibraryMacro, name)
	r.SetAttr("srcs", rule.GlobValue{Patterns: []string{
		"src/main/groovy/**/*.groovy",
	}})
	r.SetAttr("visibility", []string{gc.Visibility})

	// Parse files to collect imports and packages
	allImports, packages := g.collectImportsAndPackages(args.Dir, files)

	// Store imports for resolution phase
	r.SetPrivateAttr("groovy_imports", allImports)
	r.SetPrivateAttr("groovy_packages", packages)

	return r, allImports
}

func (g *groovyLang) generateTestRule(args language.GenerateArgs, gc *GroovyConfig, files []string, hasMain bool) (*rule.Rule, []string) {
	// Derive target name from directory name using jvm package
	baseName := jvm.DeriveTargetName(args.Dir, args.Config.RepoRoot)
	name := jvm.DeriveTestTargetName(args.Dir, args.Config.RepoRoot)

	hasSpecFiles := hasSpockSpecFiles(files)
	useSpock := shouldUseSpock(hasSpecFiles, gc)
	macro := gc.TestMacro
	if useSpock {
		macro = gc.SpockTestMacro
	}

	r := rule.NewRule(macro, name)
	if useSpock {
		specPatterns := spockSpecPatterns()
		if !hasSpecFiles {
			specPatterns = []string{testGroovyPattern}
		}
		r.SetAttr("specs", rule.GlobValue{Patterns: specPatterns})
		if hasSpecFiles {
			r.SetAttr("groovy_srcs", rule.GlobValue{
				Patterns: []string{testGroovyPattern},
				Excludes: spockSpecPatterns(),
			})
		}
	} else {
		r.SetAttr("srcs", rule.GlobValue{Patterns: []string{
			testGroovyPattern,
		}})
	}

	if hasMain {
		r.SetAttr("deps", []string{":" + baseName})
	}

	// Parse files to collect imports and packages
	allImports, packages := g.collectImportsAndPackages(args.Dir, files)

	// Store imports for resolution phase
	r.SetPrivateAttr("groovy_imports", allImports)
	r.SetPrivateAttr("groovy_packages", packages)

	return r, allImports
}

const testGroovyPattern = "src/test/groovy/**/*.groovy"

func spockSpecPatterns() []string {
	return []string{
		"src/test/groovy/**/*Spec.groovy",
		"src/test/groovy/**/*Specification.groovy",
	}
}

func hasSpockSpecFiles(files []string) bool {
	for _, rel := range files {
		base := filepath.Base(rel)
		if strings.HasSuffix(base, "Spec.groovy") || strings.HasSuffix(base, "Specification.groovy") {
			return true
		}
	}
	return false
}

func shouldUseSpock(hasSpecFiles bool, gc *GroovyConfig) bool {
	if gc.TestMacro == gc.SpockTestMacro {
		return true
	}
	if !gc.SpockDetection {
		return false
	}
	return hasSpecFiles
}

// collectImportsAndPackages parses Groovy files and collects imports and packages.
func (g *groovyLang) collectImportsAndPackages(dir string, files []string) ([]string, []string) {
	seen := make(map[string]bool)
	pkgSeen := make(map[string]bool)
	var allImports []string
	var packages []string

	for _, file := range files {
		fullPath := filepath.Join(dir, file)
		result, err := g.parser.ParseFile(fullPath)
		if err != nil {
			log.Warn("failed to parse groovy file",
				"file", file, "error", err)
			continue
		}

		// Collect package
		if result.Package != "" && !pkgSeen[result.Package] {
			pkgSeen[result.Package] = true
			packages = append(packages, result.Package)
		}

		// Collect all dependencies (imports + FQNs)
		for _, dep := range result.AllDependencies {
			if !seen[dep] {
				seen[dep] = true
				allImports = append(allImports, dep)
			}
		}

		// Also collect star imports
		for _, starImp := range result.StarImports {
			if !seen[starImp] {
				seen[starImp] = true
				allImports = append(allImports, starImp+".*")
			}
		}
	}

	return allImports, packages
}
