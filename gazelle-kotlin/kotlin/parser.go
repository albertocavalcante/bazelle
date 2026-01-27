package kotlin

import (
	"bufio"
	"maps"
	"os"
	"regexp"
	"slices"
	"strings"
)

// KotlinParser provides HEURISTIC parsing of Kotlin source files using regex.
//
// # Heuristic Parsing
//
// This parser uses regular expressions to extract metadata from Kotlin files.
// While regex cannot fully parse context-free grammars, the patterns are
// carefully designed to handle common Kotlin code conventions with high accuracy.
//
// The heuristic approach trades theoretical correctness for practical benefits:
//   - No external dependencies (pure Go)
//   - Fast parsing (single pass, minimal allocations)
//   - Sufficient for ~99% of real-world Kotlin code
//
// # Known Limitations
//
// The following edge cases may produce incorrect results:
//   - Package/import declarations inside string literals are matched as real
//   - Multi-line package declarations are not supported
//   - Unusual annotation syntax may be partially captured
//   - Comments within package/import statements may confuse the parser
//
// # Deterministic Alternative
//
// For applications requiring 100% accuracy, use TreeSitterBackend which
// provides deterministic AST-based parsing. The HeuristicBackend wraps
// this parser with the ParserBackend interface for consistency.
//
// # Thread Safety
//
// KotlinParser is safe for concurrent use. The compiled regex patterns are
// read-only after initialization, and parsing creates no shared state.
type KotlinParser struct {
	// Regex patterns for parsing (HEURISTIC - may produce false matches)
	packageRegex     *regexp.Regexp // Matches package declarations
	importRegex      *regexp.Regexp // Matches regular imports
	importAliasRegex *regexp.Regexp // Matches aliased imports (import X as Y)
	starImportRegex  *regexp.Regexp // Matches star imports (import X.*)
	annotationRegex  *regexp.Regexp // Matches @file: annotations
	declarationRegex *regexp.Regexp // Detects start of code (end of imports)

	// FQN scanner for detecting inline fully qualified names (also HEURISTIC)
	fqnScanner *FQNScanner

	// Configuration
	enableFQNScanning bool
}

// ParseResult contains the parsed metadata from a Kotlin file.
type ParseResult struct {
	// Package is the package declaration (e.g., "com.example.myapp").
	Package string

	// Imports is a list of explicit import statements.
	Imports []string

	// StarImports is a list of star imports (e.g., "com.example.*").
	StarImports []string

	// ImportAliases maps alias names to their original imports.
	// For "import com.example.Foo as Bar", this would be {"Bar": "com.example.Foo"}.
	ImportAliases map[string]string

	// FQNs is a list of fully qualified names found in the code body.
	// These are types used inline without being imported.
	FQNs []string

	// AllDependencies combines Imports and FQNs for resolution.
	AllDependencies []string

	// Annotations contains file-level annotations (e.g., "@file:JvmName").
	Annotations []string

	// FilePath is the path to the parsed file.
	FilePath string

	// CodeStartLine is the line number where code starts (after imports).
	CodeStartLine int
}

// ParserOption configures the parser.
type ParserOption func(*KotlinParser)

// WithFQNScanning enables or disables FQN scanning in the code body.
func WithFQNScanning(enabled bool) ParserOption {
	return func(p *KotlinParser) {
		p.enableFQNScanning = enabled
	}
}

// NewParser creates a new KotlinParser with the given options.
//
// The parser is configured with regex patterns optimized for common Kotlin
// code conventions. All patterns are HEURISTIC approximations of Kotlin syntax.
//
// # Regex Pattern Design Principles
//
// Each pattern is designed to:
//   - Match the most common valid syntax
//   - Avoid false positives on similar-looking constructs
//   - Fail gracefully on unusual formatting
//
// The patterns do NOT attempt to validate Kotlin syntax; they extract
// metadata that looks correct. Invalid Kotlin may still produce results.
func NewParser(opts ...ParserOption) *KotlinParser {
	p := &KotlinParser{
		// HEURISTIC: Match package declarations
		// Handles: "package com.example" and "package `reserved.keywords`"
		// Limitation: Matches in strings/comments are false positives
		packageRegex: regexp.MustCompile(`^\s*package\s+([a-zA-Z][a-zA-Z0-9_.]*|` + "`[^`]+`" + `)`),

		// HEURISTIC: Match regular imports
		// Handles: "import com.example.SomeClass"
		// Limitation: Multi-line imports with line breaks are not captured
		importRegex: regexp.MustCompile(`^\s*import\s+([a-zA-Z][a-zA-Z0-9_]*(?:\.[a-zA-Z][a-zA-Z0-9_]*)*)`),

		// HEURISTIC: Match aliased imports
		// Handles: "import com.example.SomeClass as Alias"
		// Captures: [full match, import path, alias name]
		importAliasRegex: regexp.MustCompile(`^\s*import\s+([a-zA-Z][a-zA-Z0-9_]*(?:\.[a-zA-Z][a-zA-Z0-9_]*)*)\s+as\s+(\w+)`),

		// HEURISTIC: Match star imports
		// Handles: "import com.example.*"
		// End anchor prevents matching "import com.example.*.Foo" (invalid)
		starImportRegex: regexp.MustCompile(`^\s*import\s+([a-zA-Z][a-zA-Z0-9_]*(?:\.[a-zA-Z][a-zA-Z0-9_]*)*)\.\*\s*$`),

		// HEURISTIC: Match file-level annotations
		// Handles: "@file:JvmName("Foo")", "@file:Suppress(...)"
		// Captures: annotation name only (not arguments)
		annotationRegex: regexp.MustCompile(`^\s*@file\s*:\s*(\w+)`),

		// HEURISTIC: Detect start of code (end of import section)
		// Matches any Kotlin declaration keyword at start of line
		// This determines where to stop looking for imports and start FQN scanning
		declarationRegex: regexp.MustCompile(`^\s*(class|object|interface|fun|val|var|annotation|enum|sealed|data|inline|value|suspend|private|internal|public|protected|abstract|open|expect|actual|typealias)\s`),

		fqnScanner:        NewFQNScanner(),
		enableFQNScanning: true, // enabled by default
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// ParseFile parses a Kotlin source file and returns metadata.
func (p *KotlinParser) ParseFile(path string) (*ParseResult, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return p.ParseContent(string(content), path)
}

// ParseContent parses Kotlin source code content and returns metadata.
//
// This method performs HEURISTIC parsing using regex pattern matching.
// Results are accurate for conventional Kotlin code but may be incorrect
// for edge cases. See KotlinParser documentation for known limitations.
//
// # Parsing Algorithm
//
// The parser makes a single pass through the file:
//  1. Strip block comments (/* */) and line comments (//)
//  2. Match @file: annotations before package declaration
//  3. Match package declaration
//  4. Match import statements until a declaration keyword is found
//  5. Optionally scan remaining code for FQN usage (heuristic)
//
// The CodeStartLine in the result indicates where code begins (after imports).
func (p *KotlinParser) ParseContent(content string, path string) (*ParseResult, error) {
	result := &ParseResult{
		FilePath:      path,
		Imports:       make([]string, 0),
		StarImports:   make([]string, 0),
		ImportAliases: make(map[string]string),
		FQNs:          make([]string, 0),
		Annotations:   make([]string, 0),
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	// Increase buffer size to handle very long lines (minified code, generated files)
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

		// Parse file-level annotations (before package declaration)
		if result.Package == "" && strings.HasPrefix(trimmed, "@file") {
			if matches := p.annotationRegex.FindStringSubmatch(line); len(matches) > 1 {
				result.Annotations = append(result.Annotations, matches[1])
			}
			continue
		}

		// Try to match package declaration
		if result.Package == "" {
			if matches := p.packageRegex.FindStringSubmatch(line); len(matches) > 1 {
				result.Package = cleanPackageName(matches[1])
				continue
			}
		}

		// Check if we've reached the end of imports section
		if p.declarationRegex.MatchString(line) {
			if !importSectionEnded {
				importSectionEnded = true
				result.CodeStartLine = lineNum
			}
			// Don't break - we might still want to scan for FQNs
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

		// Try to match import with alias
		if matches := p.importAliasRegex.FindStringSubmatch(line); len(matches) > 2 {
			importPath := matches[1]
			alias := matches[2]
			result.Imports = append(result.Imports, importPath)
			result.ImportAliases[alias] = importPath
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

// ParseFiles parses multiple Kotlin files and returns their metadata.
func (p *KotlinParser) ParseFiles(paths []string) ([]*ParseResult, error) {
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

// buildAllDependencies combines imports and FQNs into a single list.
func buildAllDependencies(result *ParseResult) []string {
	depSet := make(map[string]bool)

	// Add regular imports
	for _, imp := range result.Imports {
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

// cleanPackageName removes backticks from package names.
func cleanPackageName(name string) string {
	return strings.Trim(name, "`")
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

// ImportInfo contains detailed information about an import.
type ImportInfo struct {
	// Path is the full import path (e.g., "com.example.Foo").
	Path string

	// Package is the package portion (e.g., "com.example").
	Package string

	// Name is the imported name (e.g., "Foo").
	Name string

	// Alias is the alias if present, empty otherwise.
	Alias string

	// IsStar indicates if this is a star import.
	IsStar bool
}

// GetImportInfo returns detailed import information from a parse result.
func GetImportInfo(result *ParseResult) []ImportInfo {
	var infos []ImportInfo

	// Build reverse map: path -> alias (for efficient lookup)
	pathToAlias := make(map[string]string, len(result.ImportAliases))
	for alias, path := range result.ImportAliases {
		pathToAlias[path] = alias
	}

	// Regular imports
	for _, imp := range result.Imports {
		info := ImportInfo{
			Path:    imp,
			Package: ExtractPackageFromFQN(imp),
			Name:    ExtractClassFromFQN(imp),
		}
		// O(1) lookup for alias using reverse map
		if alias, ok := pathToAlias[imp]; ok {
			info.Alias = alias
		}
		infos = append(infos, info)
	}

	// Star imports
	for _, starImp := range result.StarImports {
		infos = append(infos, ImportInfo{
			Path:    starImp + ".*",
			Package: starImp,
			IsStar:  true,
		})
	}

	return infos
}
