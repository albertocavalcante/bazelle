package groovy

import (
	"bufio"
	_ "embed"
	"regexp"
	"slices"
	"strings"
	"sync"
)

// Package-level compiled regexes for removeStringLiterals (performance optimization).
// These are used to strip string content before FQN detection to reduce false positives.
var (
	stringLiteralRegex = regexp.MustCompile(`"(?:[^"\\]|\\.)*"`)
	charLiteralRegex   = regexp.MustCompile(`'(?:[^'\\]|\\.)*'`)
)

// Embedded data files for FQN filtering.
// These lists help the heuristic scanner distinguish between:
//   - Stdlib types (excluded - always available)
//   - Common third-party packages (included - need dependency tracking)
//   - Built-in types (excluded - not real dependencies)

//go:embed groovy_stdlib_prefixes.txt
var groovyStdlibPrefixesData string

//go:embed groovy_builtin_types.txt
var groovyBuiltinTypesData string

//go:embed groovy_common_prefixes.txt
var groovyCommonPrefixesData string

// Lazily initialized data from embedded files
var (
	groovyStdlibPrefixes     map[string]bool
	groovyStdlibPrefixesOnce sync.Once

	groovyBuiltinTypes     map[string]bool
	groovyBuiltinTypesOnce sync.Once

	groovyCommonPrefixes     []string
	groovyCommonPrefixesOnce sync.Once
)

// initGroovyStdlibPrefixes initializes the stdlib prefixes set from embedded data.
func initGroovyStdlibPrefixes() {
	groovyStdlibPrefixes = make(map[string]bool)
	scanner := bufio.NewScanner(strings.NewReader(groovyStdlibPrefixesData))
	for scanner.Scan() {
		prefix := strings.TrimSpace(scanner.Text())
		if prefix != "" && !strings.HasPrefix(prefix, "#") {
			groovyStdlibPrefixes[prefix] = true
		}
	}
}

// getGroovyStdlibPrefixes returns the stdlib prefixes set, initializing lazily.
func getGroovyStdlibPrefixes() map[string]bool {
	groovyStdlibPrefixesOnce.Do(initGroovyStdlibPrefixes)
	return groovyStdlibPrefixes
}

// initGroovyBuiltinTypes initializes the builtin types set from embedded data.
func initGroovyBuiltinTypes() {
	groovyBuiltinTypes = make(map[string]bool)
	scanner := bufio.NewScanner(strings.NewReader(groovyBuiltinTypesData))
	for scanner.Scan() {
		typeName := strings.TrimSpace(scanner.Text())
		if typeName != "" && !strings.HasPrefix(typeName, "#") {
			groovyBuiltinTypes[typeName] = true
		}
	}
}

// getGroovyBuiltinTypes returns the builtin types set, initializing lazily.
func getGroovyBuiltinTypes() map[string]bool {
	groovyBuiltinTypesOnce.Do(initGroovyBuiltinTypes)
	return groovyBuiltinTypes
}

// initGroovyCommonPrefixes initializes the common prefixes list from embedded data.
func initGroovyCommonPrefixes() {
	groovyCommonPrefixes = make([]string, 0)
	scanner := bufio.NewScanner(strings.NewReader(groovyCommonPrefixesData))
	for scanner.Scan() {
		prefix := strings.TrimSpace(scanner.Text())
		if prefix != "" && !strings.HasPrefix(prefix, "#") {
			groovyCommonPrefixes = append(groovyCommonPrefixes, prefix)
		}
	}
}

// getGroovyCommonPrefixes returns the common prefixes list, initializing lazily.
func getGroovyCommonPrefixes() []string {
	groovyCommonPrefixesOnce.Do(initGroovyCommonPrefixes)
	return groovyCommonPrefixes
}

// FQNScanner detects fully qualified names used inline without imports.
//
// # HEURISTIC BEHAVIOR
//
// This scanner is ALWAYS heuristic, regardless of which parser backend is used.
// True FQN detection requires semantic analysis (type resolution) that is beyond
// the scope of syntax parsing. The scanner uses pattern matching to identify
// likely FQNs based on:
//   - Known package prefixes (com., org., io., etc.)
//   - Naming conventions (lowercase.packages.UppercaseClass)
//   - Context clues (: Type, as Type, FQN())
//
// # Use Cases
//
// FQN scanning is particularly useful for:
//   - AI-generated code that uses FQNs instead of imports
//   - Migrated code with inconsistent import styles
//   - Codebases that prefer explicit FQN usage
//
// # Known Limitations
//
// The scanner may produce:
//   - False positives: Package-qualified method calls that look like FQNs
//   - False negatives: FQNs with unusual package naming conventions
//   - Missed nested classes: Only the outer class is captured
//
// # Filtering
//
// The scanner excludes:
//   - Groovy/Java stdlib types (groovy.*, java.*, javax.*, etc.) - always available
//   - Built-in types (String, Integer, List, etc.) - no dependency needed
type FQNScanner struct {
	// Patterns for common package prefixes (HEURISTIC)
	fqnPatterns []*regexp.Regexp

	// Pattern to detect type annotations and usages (HEURISTIC)
	typeUsagePattern *regexp.Regexp

	// Pattern to detect function calls on FQN (HEURISTIC)
	fqnCallPattern *regexp.Regexp

	// Known standard library packages to exclude (DETERMINISTIC lookup)
	stdlibPrefixes map[string]bool

	// Known Groovy built-in types to exclude (DETERMINISTIC lookup)
	builtinTypes map[string]bool
}

// NewFQNScanner creates a new FQN scanner with default patterns.
func NewFQNScanner() *FQNScanner {
	s := &FQNScanner{
		stdlibPrefixes: getGroovyStdlibPrefixes(),
		builtinTypes:   getGroovyBuiltinTypes(),
	}

	// Load common third-party package prefixes from embedded file.
	commonPrefixes := getGroovyCommonPrefixes()

	// Build prefix-specific FQN patterns (HEURISTIC)
	for _, prefix := range commonPrefixes {
		pattern := regexp.MustCompile(
			`\b` + regexp.QuoteMeta(prefix) + `\.` +
				`[a-z][a-zA-Z0-9_]*` +
				`(?:\.[a-z][a-zA-Z0-9_]*)*` +
				`\.([A-Z][a-zA-Z0-9_]*)`,
		)
		s.fqnPatterns = append(s.fqnPatterns, pattern)
	}

	// Generic FQN pattern for less common prefixes (HEURISTIC)
	s.fqnPatterns = append(s.fqnPatterns, regexp.MustCompile(
		`\b([a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*){2,}\.[A-Z][a-zA-Z0-9_]*)`,
	))

	// Type context patterns (HEURISTIC)
	// Matches FQNs in type positions
	s.typeUsagePattern = regexp.MustCompile(
		`(?::\s*|as\s+|instanceof\s+)([a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)+\.[A-Z][a-zA-Z0-9_]*)`,
	)

	// Function/constructor call pattern (HEURISTIC)
	s.fqnCallPattern = regexp.MustCompile(
		`\b([a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)+\.[A-Z][a-zA-Z0-9_]*)\s*[(<]`,
	)

	return s
}

// ScanResult contains FQNs found in the code body.
type ScanResult struct {
	// FQNs is a deduplicated, sorted list of fully qualified names found.
	FQNs []string

	// FQNToLocations maps each FQN to line numbers where it was found.
	FQNToLocations map[string][]int
}

// Scan scans the code body for FQN usages using HEURISTIC pattern matching.
func (s *FQNScanner) Scan(content string, codeStartLine int) *ScanResult {
	result := &ScanResult{
		FQNs:           make([]string, 0),
		FQNToLocations: make(map[string][]int),
	}

	lines := strings.Split(content, "\n")
	if codeStartLine < 0 || codeStartLine >= len(lines) {
		return result
	}

	fqnSet := make(map[string]bool)
	inTripleQuote := false
	inBlockComment := false

	addFQN := func(fqn string, lineNum int) {
		fqn = cleanFQN(fqn)
		if s.shouldInclude(fqn) {
			if !fqnSet[fqn] {
				fqnSet[fqn] = true
				result.FQNs = append(result.FQNs, fqn)
			}
			result.FQNToLocations[fqn] = append(result.FQNToLocations[fqn], lineNum+1)
		}
	}

	for lineNum := codeStartLine; lineNum < len(lines); lineNum++ {
		line := lines[lineNum]

		// Strip triple-quoted string content
		line, inTripleQuote = stripTripleQuoted(line, inTripleQuote)

		line, inBlockComment = stripComments(line, inBlockComment)

		if strings.TrimSpace(line) == "" {
			continue
		}

		// Remove string literals to avoid false positives
		line = removeStringLiterals(line)

		// Scan with all FQN patterns
		for _, pattern := range s.fqnPatterns {
			for _, match := range pattern.FindAllStringSubmatch(line, -1) {
				addFQN(match[0], lineNum)
			}
		}

		// Scan with type usage pattern
		for _, match := range s.typeUsagePattern.FindAllStringSubmatch(line, -1) {
			if len(match) > 1 {
				addFQN(match[1], lineNum)
			}
		}

		// Scan for FQN function calls
		for _, match := range s.fqnCallPattern.FindAllStringSubmatch(line, -1) {
			if len(match) > 1 {
				addFQN(match[1], lineNum)
			}
		}
	}

	// Sort FQNs for deterministic output
	slices.Sort(result.FQNs)

	return result
}

// shouldInclude determines if an FQN should be included in results.
func (s *FQNScanner) shouldInclude(fqn string) bool {
	if fqn == "" {
		return false
	}

	// Must have at least 2 dots (package.subpackage.Class)
	if strings.Count(fqn, ".") < 2 {
		return false
	}

	// Extract the class name (last segment)
	parts := strings.Split(fqn, ".")
	className := parts[len(parts)-1]

	// Class name must start with uppercase
	if len(className) == 0 || className[0] < 'A' || className[0] > 'Z' {
		return false
	}

	// Exclude built-in types
	if s.builtinTypes[className] {
		return false
	}

	// Get the prefix (first segment)
	prefix := parts[0]

	// Exclude groovy/java stdlib
	if s.stdlibPrefixes[prefix] {
		return false
	}

	return true
}

// cleanFQN removes any trailing characters that aren't part of the FQN.
func cleanFQN(fqn string) string {
	fqn = strings.TrimSpace(fqn)

	// Remove generic type parameters
	if idx := strings.Index(fqn, "<"); idx > 0 {
		fqn = fqn[:idx]
	}

	// Remove array brackets
	if idx := strings.Index(fqn, "["); idx > 0 {
		fqn = fqn[:idx]
	}

	// Remove any trailing punctuation
	fqn = strings.TrimRight(fqn, ".,;:(){}[]<>?!")

	return fqn
}

// removeStringLiterals removes string content to avoid false FQN matches.
func removeStringLiterals(line string) string {
	// Remove regular strings (handles escaped quotes)
	result := stringLiteralRegex.ReplaceAllString(line, `""`)

	// Remove char literals
	result = charLiteralRegex.ReplaceAllString(result, `''`)

	return result
}
