package groovy

import (
	"bufio"
	"maps"
	"os"
	"regexp"
	"slices"
	"strings"
)

// GroovyParser provides HEURISTIC parsing of Groovy source files using regex.
//
// # Heuristic Parsing
//
// This parser uses regular expressions to extract metadata from Groovy files.
// While regex cannot fully parse context-free grammars, the patterns are
// carefully designed to handle common Groovy code conventions with high accuracy.
//
// The heuristic approach trades theoretical correctness for practical benefits:
//   - No external dependencies (pure Go)
//   - Fast parsing (single pass, minimal allocations)
//   - Sufficient for ~99% of real-world Groovy code
//
// # Known Limitations
//
// The following edge cases may produce incorrect results:
//   - Package/import declarations inside string literals are matched as real
//   - Multi-line package declarations are not supported
//   - @Grab annotations with unusual formatting may be partially captured
//   - Comments within package/import statements may confuse the parser
//
// # Thread Safety
//
// GroovyParser is safe for concurrent use. The compiled regex patterns are
// read-only after initialization, and parsing creates no shared state.
type GroovyParser struct {
	// Regex patterns for parsing (HEURISTIC - may produce false matches)
	packageRegex      *regexp.Regexp // Matches package declarations
	importRegex       *regexp.Regexp // Matches regular imports
	importStaticRegex *regexp.Regexp // Matches static imports
	starImportRegex   *regexp.Regexp // Matches star imports (import X.*)
	grabRegex         *regexp.Regexp // Matches @Grab annotations
	declarationRegex  *regexp.Regexp // Detects start of code (end of imports)

	// FQN scanner for detecting inline fully qualified names (also HEURISTIC)
	fqnScanner *FQNScanner

	// Configuration
	enableFQNScanning bool
}

// GrabDependency represents a @Grab annotation dependency.
type GrabDependency struct {
	Group   string
	Module  string
	Version string
}

// ParseResult contains the parsed metadata from a Groovy file.
type ParseResult struct {
	// Package is the package declaration (e.g., "com.example.myapp").
	Package string

	// Imports is a list of explicit import statements.
	Imports []string

	// StaticImports is a list of static import statements.
	StaticImports []string

	// StarImports is a list of star imports (e.g., "com.example.*").
	StarImports []string

	// GrabDeps is a list of @Grab annotation dependencies.
	GrabDeps []GrabDependency

	// FQNs is a list of fully qualified names found in the code body.
	// These are types used inline without being imported.
	FQNs []string

	// AllDependencies combines Imports and FQNs for resolution.
	AllDependencies []string

	// FilePath is the path to the parsed file.
	FilePath string

	// CodeStartLine is the line number where code starts (after imports).
	CodeStartLine int
}

// ParserOption configures the parser.
type ParserOption func(*GroovyParser)

// WithFQNScanning enables or disables FQN scanning in the code body.
func WithFQNScanning(enabled bool) ParserOption {
	return func(p *GroovyParser) {
		p.enableFQNScanning = enabled
	}
}

// NewParser creates a new GroovyParser with the given options.
//
// The parser is configured with regex patterns optimized for common Groovy
// code conventions. All patterns are HEURISTIC approximations of Groovy syntax.
func NewParser(opts ...ParserOption) *GroovyParser {
	p := &GroovyParser{
		// HEURISTIC: Match package declarations
		// Handles: "package com.example" and "package com.example.app"
		// Limitation: Matches in strings/comments are false positives
		packageRegex: regexp.MustCompile(`^\s*package\s+([a-zA-Z][a-zA-Z0-9_.]*)`),

		// HEURISTIC: Match regular imports
		// Handles: "import com.example.SomeClass"
		// Limitation: Multi-line imports with line breaks are not captured
		importRegex: regexp.MustCompile(`^\s*import\s+([a-zA-Z][a-zA-Z0-9_]*(?:\.[a-zA-Z][a-zA-Z0-9_]*)*)`),

		// HEURISTIC: Match static imports
		// Handles: "import static com.example.SomeClass.method"
		// Captures: the full import path including the static member
		importStaticRegex: regexp.MustCompile(`^\s*import\s+static\s+([a-zA-Z][a-zA-Z0-9_]*(?:\.[a-zA-Z][a-zA-Z0-9_]*)*)`),

		// HEURISTIC: Match star imports
		// Handles: "import com.example.*"
		// End anchor prevents matching "import com.example.*.Foo" (invalid)
		starImportRegex: regexp.MustCompile(`^\s*import\s+([a-zA-Z][a-zA-Z0-9_]*(?:\.[a-zA-Z][a-zA-Z0-9_]*)*)\.\*\s*$`),

		// HEURISTIC: Match @Grab annotations
		// Handles: @Grab('group:module:version') and @Grab(group='x', module='y', version='z')
		// Captures: either the short form string or individual parts
		grabRegex: regexp.MustCompile(`@Grab\s*\(\s*(?:'([^']+)'|"([^"]+)"|group\s*=\s*['"]([^'"]+)['"].*module\s*=\s*['"]([^'"]+)['"].*version\s*=\s*['"]([^'"]+)['"])`),

		// HEURISTIC: Detect start of code (end of import section)
		// Matches any Groovy declaration keyword at start of line
		declarationRegex: regexp.MustCompile(`^\s*(class|interface|trait|enum|def|@|public|private|protected|abstract|final|static\s+class|static\s+interface)\s`),

		fqnScanner:        NewFQNScanner(),
		enableFQNScanning: true, // enabled by default
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// ParseFile parses a Groovy source file and returns metadata.
func (p *GroovyParser) ParseFile(path string) (*ParseResult, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return p.ParseContent(string(content), path)
}

// ParseContent parses Groovy source code content and returns metadata.
//
// This method performs HEURISTIC parsing using regex pattern matching.
// Results are accurate for conventional Groovy code but may be incorrect
// for edge cases. See GroovyParser documentation for known limitations.
//
// # Parsing Algorithm
//
// The parser makes a single pass through the file:
//  1. Strip block comments (/* */) and line comments (//)
//  2. Match package declaration
//  3. Match import statements until a declaration keyword is found
//  4. Match @Grab annotations for external dependencies
//  5. Optionally scan remaining code for FQN usage (heuristic)
//
// The CodeStartLine in the result indicates where code begins (after imports).
func (p *GroovyParser) ParseContent(content string, path string) (*ParseResult, error) {
	result := &ParseResult{
		FilePath:      path,
		Imports:       make([]string, 0),
		StaticImports: make([]string, 0),
		StarImports:   make([]string, 0),
		GrabDeps:      make([]GrabDependency, 0),
		FQNs:          make([]string, 0),
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	// Increase buffer size to handle very long lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024) // 1MB max token size
	inBlockComment := false
	lineNum := 0
	importSectionEnded := false

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		line, inBlockComment = stripComments(line, inBlockComment)

		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Check for @Grab annotations anywhere in the file
		if matches := p.grabRegex.FindStringSubmatch(line); len(matches) > 0 {
			grab := parseGrabAnnotation(matches)
			if grab.Module != "" {
				result.GrabDeps = append(result.GrabDeps, grab)
			}
		}

		// Try to match package declaration
		if result.Package == "" {
			if matches := p.packageRegex.FindStringSubmatch(line); len(matches) > 1 {
				result.Package = matches[1]
				continue
			}
		}

		// Check if we've reached the end of imports section
		if p.declarationRegex.MatchString(line) {
			if !importSectionEnded {
				importSectionEnded = true
				result.CodeStartLine = lineNum
			}
			continue
		}

		// Don't parse imports if we're past the import section
		if importSectionEnded {
			continue
		}

		// Try to match star imports first (more specific)
		if matches := p.starImportRegex.FindStringSubmatch(line); len(matches) > 1 {
			result.StarImports = append(result.StarImports, matches[1])
			continue
		}

		// Try to match static imports
		if matches := p.importStaticRegex.FindStringSubmatch(line); len(matches) > 1 {
			result.StaticImports = append(result.StaticImports, matches[1])
			continue
		}

		// Try to match regular import
		if matches := p.importRegex.FindStringSubmatch(line); len(matches) > 1 {
			result.Imports = append(result.Imports, matches[1])
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Set code start line if not already set
	if result.CodeStartLine == 0 {
		result.CodeStartLine = lineNum
	}

	// Scan for FQNs in the code body if enabled (HEURISTIC)
	if p.enableFQNScanning {
		startLine := max(result.CodeStartLine-1, 0)
		scanResult := p.fqnScanner.Scan(content, startLine)
		result.FQNs = scanResult.FQNs
	}

	// Build combined dependencies list
	result.AllDependencies = buildAllDependencies(result)

	return result, nil
}

// ParseFiles parses multiple Groovy files and returns their metadata.
func (p *GroovyParser) ParseFiles(paths []string) ([]*ParseResult, error) {
	results := make([]*ParseResult, 0, len(paths))
	for _, path := range paths {
		result, err := p.ParseFile(path)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

// parseGrabAnnotation extracts dependency info from @Grab regex matches.
func parseGrabAnnotation(matches []string) GrabDependency {
	// Short form: @Grab('group:module:version')
	shortForm := matches[1]
	if shortForm == "" {
		shortForm = matches[2] // Double quotes
	}
	if shortForm != "" {
		parts := strings.Split(shortForm, ":")
		if len(parts) >= 3 {
			return GrabDependency{
				Group:   parts[0],
				Module:  parts[1],
				Version: parts[2],
			}
		}
		if len(parts) == 2 {
			return GrabDependency{
				Group:  parts[0],
				Module: parts[1],
			}
		}
	}

	// Long form: @Grab(group='x', module='y', version='z')
	if len(matches) > 5 && matches[3] != "" && matches[4] != "" {
		return GrabDependency{
			Group:   matches[3],
			Module:  matches[4],
			Version: matches[5],
		}
	}

	return GrabDependency{}
}

// buildAllDependencies combines imports and FQNs into a single list.
func buildAllDependencies(result *ParseResult) []string {
	depSet := make(map[string]bool)

	// Add regular imports
	for _, imp := range result.Imports {
		depSet[imp] = true
	}

	// Add static imports
	for _, imp := range result.StaticImports {
		depSet[imp] = true
	}

	// Add FQNs (these are already full paths)
	for _, fqn := range result.FQNs {
		depSet[fqn] = true
	}

	// Note: Star imports are handled separately during resolution
	// since we don't know which specific classes are used

	return slices.Sorted(maps.Keys(depSet))
}

// GetPackages returns unique packages from parse results.
func GetPackages(results []*ParseResult) []string {
	pkgSet := make(map[string]bool)
	for _, r := range results {
		if r.Package != "" {
			pkgSet[r.Package] = true
		}
	}

	return slices.Sorted(maps.Keys(pkgSet))
}

// GetAllImports returns all unique imports from parse results.
func GetAllImports(results []*ParseResult) []string {
	importSet := make(map[string]bool)
	for _, r := range results {
		for _, imp := range r.Imports {
			importSet[imp] = true
		}
	}

	return slices.Sorted(maps.Keys(importSet))
}

// GetAllDependencies returns all unique dependencies (imports + FQNs) from parse results.
func GetAllDependencies(results []*ParseResult) []string {
	depSet := make(map[string]bool)
	for _, r := range results {
		for _, dep := range r.AllDependencies {
			depSet[dep] = true
		}
	}

	return slices.Sorted(maps.Keys(depSet))
}

// ExtractPackageFromFQN extracts the package portion from a fully qualified name.
// For example, "com.example.foo.Bar" returns "com.example.foo".
func ExtractPackageFromFQN(fqn string) string {
	if idx := strings.LastIndex(fqn, "."); idx > 0 {
		return fqn[:idx]
	}
	return ""
}

// ExtractClassFromFQN extracts the class name from a fully qualified name.
// For example, "com.example.foo.Bar" returns "Bar".
func ExtractClassFromFQN(fqn string) string {
	if idx := strings.LastIndex(fqn, "."); idx > 0 && idx < len(fqn)-1 {
		return fqn[idx+1:]
	}
	if strings.Contains(fqn, ".") {
		return ""
	}
	return fqn
}
