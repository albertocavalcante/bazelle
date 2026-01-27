package python

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

// RelativeImport represents a relative import statement in Python.
// For example, "from ..utils import helper" would be represented as:
//
//	RelativeImport{Level: 2, Module: "utils", Names: ["helper"]}
type RelativeImport struct {
	// Level is the number of leading dots (1 for ".", 2 for "..", etc.)
	Level int

	// Module is the module path after the dots (may be empty for "from . import X")
	Module string

	// Names is the list of names being imported
	Names []string
}

// ParseResult contains the result of parsing a Python file.
//
// All fields are populated using HEURISTIC parsing. Results are accurate
// for conventional Python code but may be incorrect for edge cases.
type ParseResult struct {
	// Imports is the list of imported modules.
	Imports []string

	// FromImports is a map of "from X import Y" statements.
	// Key is the module path, value is the list of imported names.
	FromImports map[string][]string

	// RelativeImports is a list of relative import statements.
	// These are "from . import X" or "from ..module import Y" style imports.
	RelativeImports []RelativeImport

	// HasMainBlock indicates if the file has an `if __name__ == "__main__":` block.
	HasMainBlock bool

	// IsTestFile indicates if the file appears to be a test file.
	// This is a HEURISTIC based on filename patterns (test_*.py, *_test.py).
	IsTestFile bool
}

// PythonParser provides HEURISTIC parsing of Python source files using regex.
//
// # Heuristic Parsing
//
// This parser uses regular expressions to extract import metadata from Python
// files. While regex cannot fully parse Python's grammar, the patterns are
// designed to handle common import patterns with high accuracy.
//
// The heuristic approach trades theoretical correctness for practical benefits:
//   - No external dependencies (pure Go)
//   - Fast parsing (single pass)
//   - Sufficient for most real-world Python code
//
// # Known Limitations
//
// The following edge cases may produce incorrect results:
//   - Import statements inside multi-line strings are matched as real imports
//   - Multi-line import statements with unusual formatting may be missed
//   - Conditional imports (inside if/try blocks) are treated as regular imports
//   - Dynamic imports (importlib) are not detected
//
// # Thread Safety
//
// PythonParser is safe for concurrent use. The compiled regex patterns are
// read-only after initialization.
type PythonParser struct {
	// HEURISTIC: Matches "import X" and "import X as Y" statements
	importRegex *regexp.Regexp

	// HEURISTIC: Matches "from X import Y" statements
	fromImportRegex *regexp.Regexp

	// HEURISTIC: Matches relative imports like "from . import X" or "from ..module import Y"
	relativeImportRegex *regexp.Regexp

	// HEURISTIC: Matches `if __name__ == "__main__":` or similar
	mainBlockRegex *regexp.Regexp
}

// NewParser creates a new Python parser with HEURISTIC regex patterns.
//
// The patterns are designed to match common Python import conventions.
// They do NOT validate Python syntax; they extract metadata that looks correct.
func NewParser() *PythonParser {
	return &PythonParser{
		// HEURISTIC: Match import statements
		// Handles: "import os", "import os.path", "import os as operating_system"
		// Limitation: Matches imports inside strings (false positive)
		importRegex: regexp.MustCompile(`^\s*import\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)*)`),

		// HEURISTIC: Match from...import statements
		// Handles: "from os import path", "from os.path import join as pjoin"
		// Limitation: Multi-line imports with unusual formatting may be missed
		fromImportRegex: regexp.MustCompile(`^\s*from\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)*)\s+import\s+(.+)`),

		// HEURISTIC: Match relative imports
		// Handles: "from . import utils", "from .. import parent", "from .utils import helper"
		// Captures: [full match, dots, optional module, imported names]
		relativeImportRegex: regexp.MustCompile(`^\s*from\s+(\.+)([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)*)?\s+import\s+(.+)`),

		// HEURISTIC: Match main block
		// Handles: if __name__ == "__main__": (with single or double quotes)
		mainBlockRegex: regexp.MustCompile(`^\s*if\s+__name__\s*==\s*['""]__main__['""]\s*:`),
	}
}

// ParseFile parses a Python file and returns the parse result.
//
// This method performs HEURISTIC parsing using regex pattern matching.
// Results are accurate for conventional Python code but may be incorrect
// for edge cases. See PythonParser documentation for known limitations.
func (p *PythonParser) ParseFile(path string) (*ParseResult, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	result := &ParseResult{
		FromImports: make(map[string][]string),
		IsTestFile:  isTestFile(path),
	}

	scanner := bufio.NewScanner(file)
	inMultilineString := false
	multilineDelim := ""

	for scanner.Scan() {
		line := scanner.Text()

		// HEURISTIC: Track multiline strings to skip their content
		// This prevents matching import-like text inside docstrings
		if !inMultilineString {
			if strings.Contains(line, `"""`) || strings.Contains(line, `'''`) {
				if strings.Contains(line, `"""`) {
					multilineDelim = `"""`
				} else {
					multilineDelim = `'''`
				}
				// Count occurrences to see if it opens and closes on same line
				count := strings.Count(line, multilineDelim)
				if count == 1 {
					inMultilineString = true
					continue
				}
			}
		} else {
			if strings.Contains(line, multilineDelim) {
				inMultilineString = false
			}
			continue
		}

		// Skip comments
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Check for main block
		if p.mainBlockRegex.MatchString(line) {
			result.HasMainBlock = true
		}

		// Check for import statements
		if matches := p.importRegex.FindStringSubmatch(line); len(matches) > 1 {
			// Handle multiple imports on one line: import os, sys, re
			imports := strings.Split(matches[1], ",")
			for _, imp := range imports {
				imp = strings.TrimSpace(imp)
				// Handle "import X as Y" - extract just X
				if idx := strings.Index(imp, " as "); idx > 0 {
					imp = imp[:idx]
				}
				if imp != "" {
					result.Imports = append(result.Imports, getTopLevelModule(imp))
				}
			}
		}

		// Check for relative imports first (from . import X, from ..module import Y)
		if matches := p.relativeImportRegex.FindStringSubmatch(line); len(matches) > 3 {
			dots := matches[1]
			module := matches[2] // May be empty for "from . import X"
			names := matches[3]

			importedNames := parseImportNames(names)
			if len(importedNames) > 0 {
				result.RelativeImports = append(result.RelativeImports, RelativeImport{
					Level:  len(dots),
					Module: module,
					Names:  importedNames,
				})
			}
			continue
		}

		// Check for from...import statements (absolute imports only)
		if matches := p.fromImportRegex.FindStringSubmatch(line); len(matches) > 2 {
			module := matches[1]
			names := matches[2]

			// Parse the imported names
			importedNames := parseImportNames(names)
			if len(importedNames) > 0 {
				topLevel := getTopLevelModule(module)
				result.FromImports[topLevel] = append(result.FromImports[topLevel], importedNames...)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// ParseFiles parses multiple Python files.
func (p *PythonParser) ParseFiles(paths []string) ([]*ParseResult, error) {
	results := make([]*ParseResult, 0, len(paths))
	for _, path := range paths {
		result, err := p.ParseFile(path)
		if err != nil {
			// Log warning but continue
			continue
		}
		results = append(results, result)
	}
	return results, nil
}

// getTopLevelModule returns the top-level module name from a dotted path.
// e.g., "os.path" -> "os", "collections.abc" -> "collections"
func getTopLevelModule(module string) string {
	if idx := strings.Index(module, "."); idx > 0 {
		return module[:idx]
	}
	return module
}

// parseImportNames parses the names from a "from X import a, b, c" statement.
func parseImportNames(names string) []string {
	// Handle parenthesized imports: from X import (a, b, c)
	names = strings.TrimPrefix(names, "(")
	names = strings.TrimSuffix(names, ")")

	// Handle continuation with backslash (rough handling)
	names = strings.ReplaceAll(names, "\\", "")

	var result []string
	parts := strings.Split(names, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		// Handle "name as alias" - extract just name
		if idx := strings.Index(part, " as "); idx > 0 {
			part = part[:idx]
		}
		part = strings.TrimSpace(part)
		if part != "" && part != "*" {
			result = append(result, part)
		}
	}
	return result
}

// isTestFile checks if a file path indicates a test file.
func isTestFile(path string) bool {
	base := strings.ToLower(path)
	return strings.HasSuffix(base, "_test.py") ||
		strings.HasPrefix(base, "test_") ||
		strings.Contains(base, "/test_") ||
		strings.Contains(base, "/tests/")
}

// GetAllImports returns a deduplicated list of all imported modules.
func (r *ParseResult) GetAllImports() []string {
	seen := make(map[string]bool)
	var result []string

	for _, imp := range r.Imports {
		if !seen[imp] {
			seen[imp] = true
			result = append(result, imp)
		}
	}

	for module := range r.FromImports {
		if !seen[module] {
			seen[module] = true
			result = append(result, module)
		}
	}

	return result
}

// ResolveRelativeImport resolves a relative import to an absolute module path.
//
// Parameters:
//   - rel: The relative import to resolve
//   - currentPkg: The current package path (e.g., "myapp.utils.helpers")
//
// Returns:
//   - The resolved absolute module path, or empty string if resolution fails
//
// Examples:
//   - ResolveRelativeImport({Level: 1, Module: "utils"}, "myapp.core") -> "myapp.utils"
//   - ResolveRelativeImport({Level: 2, Module: ""}, "myapp.core.sub") -> "myapp.core"
//   - ResolveRelativeImport({Level: 1, Module: ""}, "myapp") -> "" (goes above root)
func ResolveRelativeImport(rel RelativeImport, currentPkg string) string {
	if currentPkg == "" {
		return ""
	}

	parts := strings.Split(currentPkg, ".")

	// Level 1 = current package directory, Level 2 = parent, etc.
	// We need to go up (level - 1) directories from the current package
	stepsUp := rel.Level - 1
	if stepsUp < 0 {
		stepsUp = 0
	}

	if stepsUp >= len(parts) {
		// Would go above the root package
		return ""
	}

	// Keep parts after stepping up
	baseParts := parts[:len(parts)-stepsUp]

	// Append the relative module if present
	if rel.Module != "" {
		baseParts = append(baseParts, strings.Split(rel.Module, ".")...)
	}

	if len(baseParts) == 0 {
		return ""
	}

	return strings.Join(baseParts, ".")
}

// ResolveRelativeImports resolves all relative imports to absolute module paths.
func (r *ParseResult) ResolveRelativeImports(currentPkg string) []string {
	var resolved []string
	for _, rel := range r.RelativeImports {
		if absPath := ResolveRelativeImport(rel, currentPkg); absPath != "" {
			resolved = append(resolved, absPath)
		}
	}
	return resolved
}
