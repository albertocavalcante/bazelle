package kotlin

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

//go:embed kotlin_stdlib_prefixes.txt
var kotlinStdlibPrefixesData string

//go:embed kotlin_builtin_types.txt
var kotlinBuiltinTypesData string

//go:embed kotlin_common_prefixes.txt
var kotlinCommonPrefixesData string

// Lazily initialized data from embedded files
var (
	kotlinStdlibPrefixes     map[string]bool
	kotlinStdlibPrefixesOnce sync.Once

	kotlinBuiltinTypes     map[string]bool
	kotlinBuiltinTypesOnce sync.Once

	kotlinCommonPrefixes     []string
	kotlinCommonPrefixesOnce sync.Once
)

// initKotlinStdlibPrefixes initializes the stdlib prefixes set from embedded data.
func initKotlinStdlibPrefixes() {
	kotlinStdlibPrefixes = make(map[string]bool)
	scanner := bufio.NewScanner(strings.NewReader(kotlinStdlibPrefixesData))
	for scanner.Scan() {
		prefix := strings.TrimSpace(scanner.Text())
		if prefix != "" && !strings.HasPrefix(prefix, "#") {
			kotlinStdlibPrefixes[prefix] = true
		}
	}
}

// getKotlinStdlibPrefixes returns the stdlib prefixes set, initializing lazily.
func getKotlinStdlibPrefixes() map[string]bool {
	kotlinStdlibPrefixesOnce.Do(initKotlinStdlibPrefixes)
	return kotlinStdlibPrefixes
}

// initKotlinBuiltinTypes initializes the builtin types set from embedded data.
func initKotlinBuiltinTypes() {
	kotlinBuiltinTypes = make(map[string]bool)
	scanner := bufio.NewScanner(strings.NewReader(kotlinBuiltinTypesData))
	for scanner.Scan() {
		typeName := strings.TrimSpace(scanner.Text())
		if typeName != "" && !strings.HasPrefix(typeName, "#") {
			kotlinBuiltinTypes[typeName] = true
		}
	}
}

// getKotlinBuiltinTypes returns the builtin types set, initializing lazily.
func getKotlinBuiltinTypes() map[string]bool {
	kotlinBuiltinTypesOnce.Do(initKotlinBuiltinTypes)
	return kotlinBuiltinTypes
}

// initKotlinCommonPrefixes initializes the common prefixes list from embedded data.
func initKotlinCommonPrefixes() {
	kotlinCommonPrefixes = make([]string, 0)
	scanner := bufio.NewScanner(strings.NewReader(kotlinCommonPrefixesData))
	for scanner.Scan() {
		prefix := strings.TrimSpace(scanner.Text())
		if prefix != "" && !strings.HasPrefix(prefix, "#") {
			kotlinCommonPrefixes = append(kotlinCommonPrefixes, prefix)
		}
	}
}

// getKotlinCommonPrefixes returns the common prefixes list, initializing lazily.
func getKotlinCommonPrefixes() []string {
	kotlinCommonPrefixesOnce.Do(initKotlinCommonPrefixes)
	return kotlinCommonPrefixes
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
//   - Context clues (: Type, as Type, is Type, FQN())
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
//   - Kotlin stdlib types (kotlin.*, java.*, etc.) - always available
//   - Built-in types (String, Int, List, etc.) - no dependency needed
//   - kotlinx.* is explicitly INCLUDED (it's a separate dependency)
type FQNScanner struct {
	// Patterns for common package prefixes (HEURISTIC)
	// Each pattern matches FQNs starting with known prefixes
	fqnPatterns []*regexp.Regexp

	// Pattern to detect type annotations and usages (HEURISTIC)
	// Matches: ": com.example.Type", "as com.example.Type", "is com.example.Type"
	typeUsagePattern *regexp.Regexp

	// Pattern to detect function calls on FQN (HEURISTIC)
	// Matches: "com.example.Factory()", "com.example.Builder<T>("
	fqnCallPattern *regexp.Regexp

	// Known standard library packages to exclude (DETERMINISTIC lookup)
	// FQNs starting with these prefixes are filtered out
	stdlibPrefixes map[string]bool

	// Known Kotlin built-in types to exclude (DETERMINISTIC lookup)
	// Class names matching these are filtered out
	builtinTypes map[string]bool
}

// NewFQNScanner creates a new FQN scanner with default patterns.
//
// The scanner is configured with:
//   - Prefix patterns for common package namespaces (com, org, io, etc.)
//   - Type usage patterns for detecting FQNs in type contexts
//   - Function call patterns for detecting FQN constructor/method calls
//   - Exclusion lists for stdlib and built-in types
//
// All patterns are HEURISTIC and may produce false positives/negatives.
func NewFQNScanner() *FQNScanner {
	s := &FQNScanner{
		stdlibPrefixes: getKotlinStdlibPrefixes(),
		builtinTypes:   getKotlinBuiltinTypes(),
	}

	// Load common third-party package prefixes from embedded file.
	// These prefixes (com, org, io, etc.) are used to build targeted patterns
	// that match FQNs more accurately than a generic pattern.
	commonPrefixes := getKotlinCommonPrefixes()

	// Build prefix-specific FQN patterns (HEURISTIC)
	//
	// Pattern structure: prefix.package.subpackage.ClassName
	// Where: packages are lowercase, class name starts with uppercase
	//
	// Example matches:
	//   - com.example.foo.Bar
	//   - org.junit.Test
	//   - io.ktor.client.HttpClient
	for _, prefix := range commonPrefixes {
		pattern := regexp.MustCompile(
			`\b` + regexp.QuoteMeta(prefix) + `\.` +
				`[a-z][a-zA-Z0-9_]*` + // First package segment (lowercase start)
				`(?:\.[a-z][a-zA-Z0-9_]*)*` + // More package segments
				`\.([A-Z][a-zA-Z0-9_]*)`, // Class name (uppercase start)
		)
		s.fqnPatterns = append(s.fqnPatterns, pattern)
	}

	// Generic FQN pattern for less common prefixes (HEURISTIC)
	//
	// Matches any FQN with 3+ segments where the last starts uppercase.
	// More permissive but higher false positive rate.
	s.fqnPatterns = append(s.fqnPatterns, regexp.MustCompile(
		`\b([a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*){2,}\.[A-Z][a-zA-Z0-9_]*)`,
	))

	// Type context patterns (HEURISTIC)
	//
	// Matches FQNs in type positions where an import would normally be used:
	//   - val x: com.example.Type
	//   - x as com.example.Type
	//   - x is com.example.Type
	s.typeUsagePattern = regexp.MustCompile(
		`(?::\s*|as\s+|is\s+)([a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)+\.[A-Z][a-zA-Z0-9_]*)`,
	)

	// Function/constructor call pattern (HEURISTIC)
	//
	// Matches FQNs followed by ( or < indicating a call or generic instantiation:
	//   - com.example.Factory()
	//   - com.example.Builder<String>()
	s.fqnCallPattern = regexp.MustCompile(
		`\b([a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)+\.[A-Z][a-zA-Z0-9_]*)\s*[(<]`,
	)

	return s
}

// ScanResult contains FQNs found in the code body.
//
// All FQNs in this result are HEURISTIC detections. They should be treated
// as "likely dependencies" rather than "definite dependencies".
type ScanResult struct {
	// FQNs is a deduplicated, sorted list of fully qualified names found.
	// Each FQN represents a potential dependency that wasn't imported.
	FQNs []string

	// FQNToLocations maps each FQN to line numbers where it was found.
	// Useful for debugging or displaying FQN locations to users.
	// Line numbers are 1-indexed.
	FQNToLocations map[string][]int
}

// Scan scans the code body for FQN usages using HEURISTIC pattern matching.
//
// Parameters:
//   - content: The full file content (including imports section)
//   - codeStartLine: 0-indexed line number where code begins (after imports)
//
// The scanner only examines lines from codeStartLine onwards, avoiding
// false matches in the import section.
//
// # Algorithm
//
// For each line after codeStartLine:
//  1. Strip triple-quoted strings ("""...""")
//  2. Strip comments (// and /* */)
//  3. Remove string literals to avoid false matches
//  4. Apply all FQN patterns to find matches
//  5. Filter results through shouldInclude() to remove stdlib/builtins
//
// # Thread Safety
//
// Scan is safe for concurrent use. It creates no shared mutable state.
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

	// addFQN is a helper to deduplicate and track FQN locations
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

		// Strip triple-quoted string content while tracking multi-line state.
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

		// Scan with type usage pattern (: Type, as Type, is Type)
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
//
// This method applies DETERMINISTIC filtering based on known lists:
//   - Excludes empty or malformed FQNs
//   - Excludes FQNs with fewer than 3 segments (not specific enough)
//   - Excludes Kotlin/Java stdlib types (always on classpath)
//   - Excludes built-in type names (String, Int, etc.)
//   - Explicitly INCLUDES kotlinx.* (separate dependency from stdlib)
//
// The filtering is deterministic given the same exclusion lists, but the
// lists themselves are heuristic choices about what constitutes "stdlib".
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

	// Include kotlinx explicitly (it's a separate dependency from kotlin stdlib)
	if prefix == "kotlinx" {
		return true
	}

	// Exclude kotlin/java stdlib (they're usually already on classpath)
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

	// Remove nullable marker
	fqn = strings.TrimSuffix(fqn, "?")

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

// stripTripleQuoted removes triple-quoted string content from a single line.
// It returns the line with string content removed and the updated inTripleQuote state.
func stripTripleQuoted(line string, inTripleQuote bool) (string, bool) {
	if !inTripleQuote && !strings.Contains(line, `"""`) {
		return line, false
	}

	var out strings.Builder
	i := 0

	for i < len(line) {
		if inTripleQuote {
			end := strings.Index(line[i:], `"""`)
			if end == -1 {
				// Entire remainder is inside a triple-quoted string.
				return out.String(), true
			}
			i += end + 3
			inTripleQuote = false
			continue
		}

		start := strings.Index(line[i:], `"""`)
		if start == -1 {
			out.WriteString(line[i:])
			return out.String(), false
		}
		out.WriteString(line[i : i+start])
		i += start + 3
		inTripleQuote = true
	}

	return out.String(), inTripleQuote
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
// Returns empty string for malformed FQNs like "com." or ".".
func ExtractClassFromFQN(fqn string) string {
	if idx := strings.LastIndex(fqn, "."); idx > 0 && idx < len(fqn)-1 {
		return fqn[idx+1:]
	}
	// Return empty for malformed FQNs (trailing dot, only dot, etc.)
	if strings.Contains(fqn, ".") {
		return ""
	}
	// No dot - return the FQN itself (simple class name)
	return fqn
}
