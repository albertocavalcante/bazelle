package groovy

import (
	"path/filepath"
	"strings"

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
		libRule := g.generateLibraryRule(args, gc)
		if libRule != nil {
			rules = append(rules, libRule)
			imports = append(imports, nil)
		}
	}

	if len(testFiles) > 0 {
		testRule := g.generateTestRule(args, gc, testFiles, len(mainFiles) > 0)
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


func (g *groovyLang) generateLibraryRule(args language.GenerateArgs, gc *GroovyConfig) *rule.Rule {
	// Derive target name from directory name using jvm package
	name := jvm.DeriveTargetName(args.Dir, args.Config.RepoRoot)

	r := rule.NewRule(gc.LibraryMacro, name)
	r.SetAttr("srcs", rule.GlobValue{Patterns: []string{
		"src/main/groovy/**/*.groovy",
	}})
	r.SetAttr("visibility", []string{gc.Visibility})

	return r
}

func (g *groovyLang) generateTestRule(args language.GenerateArgs, gc *GroovyConfig, files []string, hasMain bool) *rule.Rule {
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

	return r
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
