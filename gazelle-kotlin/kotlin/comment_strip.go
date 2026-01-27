package kotlin

import "strings"

// stripComments removes line (//) and block (/* */) comments from a line,
// while preserving content inside string literals.
//
// This is a HEURISTIC helper used by both KotlinParser and FQNScanner.
// It helps reduce false positives by removing commented-out code before
// pattern matching.
//
// # String Handling
//
// The function tracks whether we're inside a double-quoted string and
// preserves comment-like characters within strings. For example:
//
//	val url = "https://example.com"  // comment -> preserves the URL
//	val x = "/* not a comment */"    // works correctly
//
// # Limitations
//
// The implementation is still approximate:
//   - Single-quoted char literals with // or /* may cause issues
//   - Escaped quotes inside strings are handled but edge cases may exist
//   - Triple-quoted raw strings (""") are NOT handled (use stripTripleQuoted first)
//   - Nested block comments are not supported (/* /* */ */ fails)
//
// For fully accurate comment stripping, use tree-sitter AST parsing.
//
// # Parameters
//
//   - line: A single line of source code
//   - inBlock: True if we're currently inside a block comment from previous lines
//
// # Returns
//
//   - The line with comments removed
//   - Whether we're still inside a block comment after this line
func stripComments(line string, inBlock bool) (string, bool) {
	var out strings.Builder
	i := 0
	inString := false

	for i < len(line) {
		// Handle block comment state first
		if inBlock {
			end := strings.Index(line[i:], "*/")
			if end == -1 {
				return out.String(), true
			}
			i += end + 2
			inBlock = false
			continue
		}

		ch := line[i]

		// Track string literals to avoid false positives
		if ch == '"' && !inString {
			// Start of string
			inString = true
			out.WriteByte(ch)
			i++
			continue
		}

		if inString {
			out.WriteByte(ch)
			if ch == '\\' && i+1 < len(line) {
				// Escaped character - write next char too and skip it
				i++
				out.WriteByte(line[i])
			} else if ch == '"' {
				// End of string
				inString = false
			}
			i++
			continue
		}

		// Not in string - check for comments
		if strings.HasPrefix(line[i:], "/*") {
			inBlock = true
			i += 2
			continue
		}
		if strings.HasPrefix(line[i:], "//") {
			// Line comment - stop processing
			break
		}

		out.WriteByte(ch)
		i++
	}

	return out.String(), inBlock
}
