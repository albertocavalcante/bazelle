package groovy

import "strings"

// stripComments removes line (//) and block (/* */) comments from a line,
// while preserving content inside string literals.
//
// This is a HEURISTIC helper used by both GroovyParser and FQNScanner.
// It helps reduce false positives by removing commented-out code before
// pattern matching.
//
// # String Handling
//
// The function tracks whether we're inside a double-quoted string and
// preserves comment-like characters within strings. For example:
//
//	def url = "https://example.com"  // comment -> preserves the URL
//	def x = "/* not a comment */"    // works correctly
//
// # Limitations
//
// The implementation is still approximate:
//   - Single-quoted char literals with // or /* may cause issues
//   - Escaped quotes inside strings are handled but edge cases may exist
//   - Triple-quoted raw strings (""" or ''') are NOT handled (use stripTripleQuoted first)
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

// stripTripleQuoted removes triple-quoted string content from a single line.
// It returns the line with string content removed and the updated inTripleQuote state.
//
// Groovy supports both """ and ''' for multiline strings.
func stripTripleQuoted(line string, inTripleQuote bool) (string, bool) {
	if !inTripleQuote && !strings.Contains(line, `"""`) && !strings.Contains(line, `'''`) {
		return line, false
	}

	var out strings.Builder
	i := 0

	for i < len(line) {
		if inTripleQuote {
			// Look for end of triple-quoted string
			if strings.HasPrefix(line[i:], `"""`) || strings.HasPrefix(line[i:], `'''`) {
				i += 3
				inTripleQuote = false
				continue
			}
			i++
			continue
		}

		// Check for start of triple-quoted string
		if strings.HasPrefix(line[i:], `"""`) || strings.HasPrefix(line[i:], `'''`) {
			i += 3
			inTripleQuote = true
			continue
		}

		out.WriteByte(line[i])
		i++
	}

	return out.String(), inTripleQuote
}
