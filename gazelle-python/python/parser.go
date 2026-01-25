package python

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

// ParseResult contains the result of parsing a Python file.
type ParseResult struct {
	// Imports is the list of imported modules.
	Imports []string

	// FromImports is a map of "from X import Y" statements.
	// Key is the module path, value is the list of imported names.
	FromImports map[string][]string

	// HasMainBlock indicates if the file has an `if __name__ == "__main__":` block.
	HasMainBlock bool

	// IsTestFile indicates if the file appears to be a test file.
	IsTestFile bool
}

// PythonParser parses Python files to extract import information.
type PythonParser struct {
	// importRegex matches "import X" and "import X as Y" statements
	importRegex *regexp.Regexp

	// fromImportRegex matches "from X import Y" statements
	fromImportRegex *regexp.Regexp

	// mainBlockRegex matches `if __name__ == "__main__":` or similar
	mainBlockRegex *regexp.Regexp
}

// NewParser creates a new Python parser.
func NewParser() *PythonParser {
	return &PythonParser{
		// Match: import module, import module as alias, import module1, module2
		importRegex: regexp.MustCompile(`^\s*import\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)*)`),

		// Match: from module import name, from module import name as alias
		fromImportRegex: regexp.MustCompile(`^\s*from\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)*)\s+import\s+(.+)`),

		// Match: if __name__ == "__main__": or if __name__ == '__main__':
		mainBlockRegex: regexp.MustCompile(`^\s*if\s+__name__\s*==\s*['""]__main__['""]\s*:`),
	}
}

// ParseFile parses a Python file and returns the parse result.
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

		// Track multiline strings (rough heuristic)
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

		// Check for from...import statements
		if matches := p.fromImportRegex.FindStringSubmatch(line); len(matches) > 2 {
			module := matches[1]
			names := matches[2]

			// Handle relative imports (from . import X, from .. import X)
			if strings.HasPrefix(module, ".") {
				// Skip relative imports for now - they're internal
				continue
			}

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
