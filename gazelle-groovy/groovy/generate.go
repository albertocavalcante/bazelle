package groovy

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// GenerateRules implements language.Language.
func (g *groovyLang) GenerateRules(args language.GenerateArgs) language.GenerateResult {
	gc := GetGroovyConfig(args.Config)
	if !gc.Enabled {
		return language.GenerateResult{}
	}

	mainFiles := findGroovyFiles(args.Dir, "src/main/groovy")
	testFiles := findGroovyFiles(args.Dir, "src/test/groovy")

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

// findGroovyFiles finds all .groovy files under a subdirectory.
func findGroovyFiles(baseDir, subDir string) []string {
	dir := filepath.Join(baseDir, subDir)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}

	var files []string
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, ".groovy") {
			relPath, _ := filepath.Rel(baseDir, path)
			files = append(files, relPath)
		}
		return nil
	})

	return files
}

func (g *groovyLang) generateLibraryRule(args language.GenerateArgs, gc *GroovyConfig) *rule.Rule {
	name := filepath.Base(args.Dir)
	if name == "." || name == "" {
		name = filepath.Base(args.Config.RepoRoot)
	}

	r := rule.NewRule(gc.LibraryMacro, name)
	r.SetAttr("srcs", rule.GlobValue{Patterns: []string{
		"src/main/groovy/**/*.groovy",
	}})
	r.SetAttr("visibility", []string{gc.Visibility})

	return r
}

func (g *groovyLang) generateTestRule(args language.GenerateArgs, gc *GroovyConfig, files []string, hasMain bool) *rule.Rule {
	baseName := filepath.Base(args.Dir)
	if baseName == "." || baseName == "" {
		baseName = filepath.Base(args.Config.RepoRoot)
	}
	name := baseName + "_test"

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
