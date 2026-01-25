package python

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/albertocavalcante/bazelle/internal/log"
	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// GenerateRules implements language.Language.
func (p *pythonLang) GenerateRules(args language.GenerateArgs) language.GenerateResult {
	pc := GetPythonConfig(args.Config)

	if !pc.Enabled {
		return language.GenerateResult{}
	}

	// Find Python source files
	mainFiles := findPythonSources(args.Dir, false)
	testFiles := findPythonSources(args.Dir, true)

	if len(mainFiles) == 0 && len(testFiles) == 0 {
		return language.GenerateResult{}
	}

	var rules []*rule.Rule
	var imports []any

	// Generate library rule for main sources
	if len(mainFiles) > 0 {
		libRule, libImports := p.generateLibraryRule(args, pc, mainFiles)
		if libRule != nil {
			rules = append(rules, libRule)
			imports = append(imports, libImports)
		}
	}

	// Generate test rule for test sources
	if len(testFiles) > 0 {
		testRule, testImports := p.generateTestRule(args, pc, testFiles, len(mainFiles) > 0)
		if testRule != nil {
			rules = append(rules, testRule)
			imports = append(imports, testImports)
		}
	}

	// Check for files with main blocks (potential binaries)
	for _, file := range mainFiles {
		fullPath := filepath.Join(args.Dir, file)
		result, err := p.parser.ParseFile(fullPath)
		if err != nil {
			continue
		}
		if result.HasMainBlock {
			binRule, binImports := p.generateBinaryRule(args, pc, file)
			if binRule != nil {
				rules = append(rules, binRule)
				imports = append(imports, binImports)
			}
		}
	}

	return language.GenerateResult{
		Gen:     rules,
		Imports: imports,
	}
}

// generateLibraryRule creates a py_library (or custom macro) rule.
func (p *pythonLang) generateLibraryRule(args language.GenerateArgs, pc *PythonConfig, files []string) (*rule.Rule, []string) {
	// Derive target name from directory name
	name := deriveTargetName(args.Dir, args.Config.RepoRoot)

	r := rule.NewRule(pc.LibraryMacro, name)
	r.SetAttr("srcs", getSrcGlobs(files))
	r.SetAttr("visibility", []string{pc.Visibility})

	// Parse files to collect imports
	allImports := p.collectImports(args.Dir, files)

	// Store imports for resolution phase
	r.SetPrivateAttr("python_imports", allImports)

	return r, allImports
}

// generateTestRule creates a py_test (or custom macro) rule.
func (p *pythonLang) generateTestRule(args language.GenerateArgs, pc *PythonConfig, files []string, hasMain bool) (*rule.Rule, []string) {
	baseName := deriveTargetName(args.Dir, args.Config.RepoRoot)
	name := baseName + "_test"

	r := rule.NewRule(pc.TestMacro, name)
	r.SetAttr("srcs", getTestSrcGlobs())

	// Parse files to collect imports
	allImports := p.collectImports(args.Dir, files)

	// Store imports for resolution phase
	r.SetPrivateAttr("python_imports", allImports)

	// Add dependency on the library if it exists
	if hasMain {
		r.SetAttr("deps", []string{":" + baseName})
	}

	return r, allImports
}

// generateBinaryRule creates a py_binary (or custom macro) rule.
func (p *pythonLang) generateBinaryRule(args language.GenerateArgs, pc *PythonConfig, mainFile string) (*rule.Rule, []string) {
	// Derive name from the file name (without .py extension)
	name := strings.TrimSuffix(mainFile, ".py")
	name = strings.ReplaceAll(name, "/", "_")

	r := rule.NewRule(pc.BinaryMacro, name)
	r.SetAttr("srcs", []string{mainFile})
	r.SetAttr("main", mainFile)

	// Parse file to collect imports
	fullPath := filepath.Join(args.Dir, mainFile)
	result, err := p.parser.ParseFile(fullPath)
	if err != nil {
		log.Warn("failed to parse python file",
			"file", mainFile, "error", err)
		return r, nil
	}

	allImports := result.GetAllImports()
	r.SetPrivateAttr("python_imports", allImports)

	return r, allImports
}

// collectImports parses files and collects all unique imports.
func (p *pythonLang) collectImports(dir string, files []string) []string {
	seen := make(map[string]bool)
	var allImports []string

	for _, file := range files {
		fullPath := filepath.Join(dir, file)
		result, err := p.parser.ParseFile(fullPath)
		if err != nil {
			log.Warn("failed to parse python file",
				"file", file, "error", err)
			continue
		}

		for _, imp := range result.GetAllImports() {
			if !seen[imp] {
				seen[imp] = true
				allImports = append(allImports, imp)
			}
		}
	}

	return allImports
}

// findPythonSources finds Python source files in a directory.
func findPythonSources(dir string, testsOnly bool) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".py") {
			continue
		}
		// Skip __init__.py for source collection (handled separately)
		if name == "__init__.py" {
			continue
		}

		isTest := isTestFile(name)
		if testsOnly && isTest {
			files = append(files, name)
		} else if !testsOnly && !isTest {
			files = append(files, name)
		}
	}

	return files
}

// deriveTargetName derives a target name from the directory path.
func deriveTargetName(dir, repoRoot string) string {
	rel, err := filepath.Rel(repoRoot, dir)
	if err != nil {
		return filepath.Base(dir)
	}
	// Use last path segment as the name
	name := filepath.Base(rel)
	if name == "." || name == "" {
		return "lib"
	}
	// Replace hyphens with underscores for valid Python identifiers
	name = strings.ReplaceAll(name, "-", "_")
	return name
}

// getSrcGlobs returns glob patterns for Python source files.
func getSrcGlobs(files []string) rule.GlobValue {
	// Use a simple glob pattern for Python files, excluding tests
	return rule.GlobValue{
		Patterns: []string{"*.py"},
		Excludes: []string{"*_test.py", "test_*.py"},
	}
}

// getTestSrcGlobs returns glob patterns for Python test files.
func getTestSrcGlobs() rule.GlobValue {
	return rule.GlobValue{
		Patterns: []string{"*_test.py", "test_*.py"},
	}
}
