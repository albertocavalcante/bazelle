package python

// Known limitations - TODOs for future implementation:
//
// TODO(bazelle): Add pip dependency resolution from requirements.txt
//   - Parse requirements.txt to identify third-party dependencies
//   - Map pip package names to Bazel targets via rules_python pip integration
//
// TODO(bazelle): Add type stub (.pyi) file handling
//   - Detect .pyi files alongside .py files
//   - Generate appropriate rules for type stubs
//
// TODO(bazelle): Support relative imports in dependency resolution
//   - Currently relative imports are skipped
//   - Need to resolve relative imports to absolute package paths
//
// TODO(bazelle): Add namespace package support
//   - Detect namespace packages (missing __init__.py)
//   - Handle implicit namespace packages properly
//
// TODO(bazelle): Auto-update requirements.txt
//   - Detect new third-party imports and suggest additions

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
	r.SetAttr("srcs", getSrcGlobs())
	r.SetAttr("visibility", []string{pc.Visibility})

	// Handle namespace packages (PEP 420)
	if pc.NamespacePackages && isNamespacePackage(args.Dir) {
		r.SetAttr("imports", []string{"."})
	}

	// Parse files to collect imports
	allImports := p.collectImports(args, files)

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
	allImports := p.collectImports(args, files)

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

	// Collect both absolute and resolved relative imports
	allImports := result.GetAllImports()
	currentPkg := derivePythonPackage(args.Dir, args.Config.RepoRoot)
	allImports = append(allImports, result.ResolveRelativeImports(currentPkg)...)

	r.SetPrivateAttr("python_imports", allImports)

	return r, allImports
}

// collectImports parses files and collects all unique imports.
func (p *pythonLang) collectImports(args language.GenerateArgs, files []string) []string {
	seen := make(map[string]bool)
	var allImports []string

	// Derive the current Python package from the directory path
	currentPkg := derivePythonPackage(args.Dir, args.Config.RepoRoot)

	for _, file := range files {
		fullPath := filepath.Join(args.Dir, file)
		result, err := p.parser.ParseFile(fullPath)
		if err != nil {
			log.Warn("failed to parse python file",
				"file", file, "error", err)
			continue
		}

		// Collect absolute imports
		for _, imp := range result.GetAllImports() {
			if !seen[imp] {
				seen[imp] = true
				allImports = append(allImports, imp)
			}
		}

		// Resolve and collect relative imports
		for _, resolved := range result.ResolveRelativeImports(currentPkg) {
			if !seen[resolved] {
				seen[resolved] = true
				allImports = append(allImports, resolved)
			}
		}
	}

	return allImports
}

// derivePythonPackage derives a Python package name from a directory path.
// For example, "src/myapp/utils" -> "myapp.utils" (assuming src is root)
func derivePythonPackage(dir, repoRoot string) string {
	rel, err := filepath.Rel(repoRoot, dir)
	if err != nil {
		return ""
	}

	// Skip common source directories
	rel = strings.TrimPrefix(rel, "src/")
	rel = strings.TrimPrefix(rel, "lib/")
	rel = strings.TrimPrefix(rel, "python/")

	if rel == "." || rel == "" {
		return ""
	}

	// Convert path separators to dots
	pkg := strings.ReplaceAll(rel, string(filepath.Separator), ".")
	return pkg
}

// findPythonSources finds Python source files in a directory.
// Returns both .py and .pyi (type stub) files.
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

		// Accept both .py and .pyi files
		isPy := strings.HasSuffix(name, ".py")
		isPyi := strings.HasSuffix(name, ".pyi")
		if !isPy && !isPyi {
			continue
		}

		// Skip __init__.py and __init__.pyi for source collection (handled separately)
		if name == "__init__.py" || name == "__init__.pyi" {
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

// isNamespacePackage checks if a directory is a namespace package (PEP 420).
// A namespace package is a directory that contains Python files but no __init__.py.
func isNamespacePackage(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}

	hasPyFiles := false
	hasInit := false

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "__init__.py" || name == "__init__.pyi" {
			hasInit = true
		}
		if strings.HasSuffix(name, ".py") && name != "__init__.py" {
			hasPyFiles = true
		}
	}

	return hasPyFiles && !hasInit
}

// hasTypeStubs checks if a directory contains type stub files (.pyi).
func hasTypeStubs(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".pyi") {
			return true
		}
	}

	return false
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
// Includes both .py and .pyi (type stub) files.
func getSrcGlobs() rule.GlobValue {
	// Use glob patterns for Python files, excluding tests
	return rule.GlobValue{
		Patterns: []string{"*.py", "*.pyi"},
		Excludes: []string{"*_test.py", "test_*.py"},
	}
}

// getTestSrcGlobs returns glob patterns for Python test files.
func getTestSrcGlobs() rule.GlobValue {
	return rule.GlobValue{
		Patterns: []string{"*_test.py", "test_*.py"},
	}
}
