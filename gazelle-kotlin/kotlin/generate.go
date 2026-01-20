package kotlin

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// GenerateRules implements language.Language.
func (k *kotlinLang) GenerateRules(args language.GenerateArgs) language.GenerateResult {
	kc := GetKotlinConfig(args.Config)

	if !kc.Enabled {
		return language.GenerateResult{}
	}

	// Find Kotlin source files
	mainFiles := findKotlinFiles(args.Dir, "src/main/kotlin")
	testFiles := findKotlinFiles(args.Dir, "src/test/kotlin")

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
		testRule := k.generateTestRule(args, kc, testFiles)
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
	// Derive target name from directory name
	name := filepath.Base(args.Dir)
	if name == "." || name == "" {
		name = filepath.Base(args.Config.RepoRoot)
	}

	r := rule.NewRule(kc.LibraryMacro, name)
	r.SetAttr("srcs", []string{"glob([\"src/main/kotlin/**/*.kt\"])"})
	r.SetAttr("visibility", []string{kc.Visibility})

	// Parse files to get package info
	fullPaths := make([]string, len(files))
	for i, f := range files {
		fullPaths[i] = filepath.Join(args.Dir, f)
	}

	results, err := k.parser.ParseFiles(fullPaths)
	if err == nil && len(results) > 0 {
		r.SetPrivateAttr("packages", GetPackages(results))
	}

	return r
}

// generateTestRule creates a kt_jvm_test (or custom macro) rule.
func (k *kotlinLang) generateTestRule(args language.GenerateArgs, kc *KotlinConfig, files []string) *rule.Rule {
	// Derive target name from directory name
	baseName := filepath.Base(args.Dir)
	if baseName == "." || baseName == "" {
		baseName = filepath.Base(args.Config.RepoRoot)
	}
	name := baseName + "_test"

	r := rule.NewRule(kc.TestMacro, name)
	r.SetAttr("srcs", []string{"glob([\"src/test/kotlin/**/*.kt\"])"})

	// Parse files to get test packages
	fullPaths := make([]string, len(files))
	for i, f := range files {
		fullPaths[i] = filepath.Join(args.Dir, f)
	}

	results, err := k.parser.ParseFiles(fullPaths)
	if err == nil && len(results) > 0 {
		packages := GetPackages(results)
		sort.Strings(packages)
		if len(packages) > 0 {
			r.SetAttr("test_packages", packages)
		}
	}

	// Add associate to the library
	r.SetAttr("associates", []string{":" + baseName})

	// Add library as dep
	r.SetAttr("deps", []string{":" + baseName})

	return r
}

// findKotlinFiles finds all .kt files under a subdirectory.
func findKotlinFiles(baseDir, subDir string) []string {
	dir := filepath.Join(baseDir, subDir)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}

	var files []string
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, ".kt") {
			relPath, _ := filepath.Rel(baseDir, path)
			files = append(files, relPath)
		}
		return nil
	})

	return files
}
