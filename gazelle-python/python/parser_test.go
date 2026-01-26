package python

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFile(t *testing.T) {
	// Create a temporary Python file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_example.py")

	content := `
import os
import sys
from collections import defaultdict, OrderedDict
from typing import List, Dict
from pathlib import Path
import json as j

# This is a comment
def main():
    pass

if __name__ == "__main__":
    main()
`
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parser := NewParser()
	result, err := parser.ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Check imports
	expectedImports := []string{"os", "sys", "json"}
	for _, expected := range expectedImports {
		found := false
		for _, imp := range result.Imports {
			if imp == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected import %q not found in %v", expected, result.Imports)
		}
	}

	// Check from imports
	if _, ok := result.FromImports["collections"]; !ok {
		t.Error("expected 'collections' in FromImports")
	}
	if _, ok := result.FromImports["typing"]; !ok {
		t.Error("expected 'typing' in FromImports")
	}
	if _, ok := result.FromImports["pathlib"]; !ok {
		t.Error("expected 'pathlib' in FromImports")
	}

	// Check main block detection
	if !result.HasMainBlock {
		t.Error("expected HasMainBlock to be true")
	}

	// Check test file detection
	if !result.IsTestFile {
		t.Error("expected IsTestFile to be true for test_example.py")
	}
}

func TestGetAllImports(t *testing.T) {
	result := &ParseResult{
		Imports: []string{"os", "sys"},
		FromImports: map[string][]string{
			"collections": {"defaultdict"},
			"typing":      {"List"},
		},
	}

	allImports := result.GetAllImports()

	expected := []string{"os", "sys", "collections", "typing"}
	if len(allImports) != len(expected) {
		t.Errorf("expected %d imports, got %d", len(expected), len(allImports))
	}

	// Check all expected imports are present
	importSet := make(map[string]bool)
	for _, imp := range allImports {
		importSet[imp] = true
	}
	for _, exp := range expected {
		if !importSet[exp] {
			t.Errorf("expected import %q not found", exp)
		}
	}
}

func TestGetTopLevelModule(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"os", "os"},
		{"os.path", "os"},
		{"collections.abc", "collections"},
		{"urllib.parse", "urllib"},
	}

	for _, tt := range tests {
		result := getTopLevelModule(tt.input)
		if result != tt.expected {
			t.Errorf("getTopLevelModule(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestIsTestFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"test_example.py", true},
		{"example_test.py", true},
		{"tests/test_foo.py", true},
		{"example.py", false},
		{"my_module.py", false},
	}

	for _, tt := range tests {
		result := isTestFile(tt.path)
		if result != tt.expected {
			t.Errorf("isTestFile(%q) = %v, want %v", tt.path, result, tt.expected)
		}
	}
}

func TestParseImportNames(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"foo", []string{"foo"}},
		{"foo, bar, baz", []string{"foo", "bar", "baz"}},
		{"foo as f, bar as b", []string{"foo", "bar"}},
		{"(foo, bar)", []string{"foo", "bar"}},
		{"*", []string{}},
	}

	for _, tt := range tests {
		result := parseImportNames(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("parseImportNames(%q) = %v, want %v", tt.input, result, tt.expected)
			continue
		}
		for i, exp := range tt.expected {
			if result[i] != exp {
				t.Errorf("parseImportNames(%q)[%d] = %q, want %q", tt.input, i, result[i], exp)
			}
		}
	}
}

func TestParseFileWithRelativeImports(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "module.py")

	content := `
from . import sibling
from .. import parent
from .utils import helper
import external_package
`
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parser := NewParser()
	result, err := parser.ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Relative imports should be skipped
	if len(result.FromImports) > 0 {
		t.Errorf("expected no from imports (relative imports should be skipped), got %v", result.FromImports)
	}

	// But the regular import should be captured
	if len(result.Imports) != 1 || result.Imports[0] != "external_package" {
		t.Errorf("expected ['external_package'], got %v", result.Imports)
	}
}

// ============================================================================
// Additional Parser Edge Case Tests
// ============================================================================

func TestParseFileEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.py")

	content := ``
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parser := NewParser()
	result, err := parser.ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if len(result.Imports) != 0 {
		t.Errorf("expected no imports for empty file, got %v", result.Imports)
	}
	if len(result.FromImports) != 0 {
		t.Errorf("expected no from imports for empty file, got %v", result.FromImports)
	}
	if result.HasMainBlock {
		t.Error("expected HasMainBlock to be false for empty file")
	}
}

func TestParseFileOnlyComments(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "comments.py")

	content := `# This is a comment
# Another comment
# import os  -- this should NOT be parsed as an import
# from typing import List  -- neither should this
`
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parser := NewParser()
	result, err := parser.ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if len(result.Imports) != 0 {
		t.Errorf("expected no imports for comments-only file, got %v", result.Imports)
	}
	if len(result.FromImports) != 0 {
		t.Errorf("expected no from imports for comments-only file, got %v", result.FromImports)
	}
}

func TestParseFileMultilineStringTripleDoubleQuotes(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "multiline.py")

	content := `import os

"""
This is a multiline string.
import sys  # This should NOT be parsed as an import
from typing import List  # Neither should this
"""

import json
`
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parser := NewParser()
	result, err := parser.ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Should have os and json, but NOT sys or typing (inside multiline string)
	expectedImports := map[string]bool{"os": true, "json": true}
	for _, imp := range result.Imports {
		if !expectedImports[imp] {
			t.Errorf("unexpected import %q found", imp)
		}
		delete(expectedImports, imp)
	}
	if len(expectedImports) > 0 {
		t.Errorf("missing expected imports: %v", expectedImports)
	}

	// Should NOT have typing from the multiline string
	if _, ok := result.FromImports["typing"]; ok {
		t.Error("typing should not be in FromImports (was inside multiline string)")
	}
}

func TestParseFileMultilineStringSingleQuotes(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "multiline_single.py")

	content := `import os

'''
This is a multiline string with single quotes.
import sys  # This should NOT be parsed as an import
'''

import json
`
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parser := NewParser()
	result, err := parser.ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Should have os and json, but NOT sys
	foundOS := false
	foundJSON := false
	for _, imp := range result.Imports {
		if imp == "os" {
			foundOS = true
		}
		if imp == "json" {
			foundJSON = true
		}
		if imp == "sys" {
			t.Error("sys should not be imported (was inside multiline string)")
		}
	}
	if !foundOS {
		t.Error("expected 'os' import not found")
	}
	if !foundJSON {
		t.Error("expected 'json' import not found")
	}
}

func TestParseFileMultilineStringOnSameLine(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "sameline.py")

	content := `import os
docstring = """This is a single line docstring with triple quotes"""
import json
`
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parser := NewParser()
	result, err := parser.ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Both os and json should be imported
	foundOS := false
	foundJSON := false
	for _, imp := range result.Imports {
		if imp == "os" {
			foundOS = true
		}
		if imp == "json" {
			foundJSON = true
		}
	}
	if !foundOS {
		t.Error("expected 'os' import not found")
	}
	if !foundJSON {
		t.Error("expected 'json' import not found")
	}
}

func TestParseFileMainBlockVariants(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		hasMain   bool
	}{
		{
			name:    "double_quotes",
			content: `if __name__ == "__main__":`,
			hasMain: true,
		},
		{
			name:    "single_quotes",
			content: `if __name__ == '__main__':`,
			hasMain: true,
		},
		{
			name:    "with_spaces",
			content: `if __name__   ==   "__main__"  :`,
			hasMain: true,
		},
		{
			name:    "indented",
			content: `    if __name__ == "__main__":`,
			hasMain: true,
		},
		{
			name:    "no_main_block",
			content: `def main(): pass`,
			hasMain: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "main.py")

			if err := os.WriteFile(testFile, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			parser := NewParser()
			result, err := parser.ParseFile(testFile)
			if err != nil {
				t.Fatalf("ParseFile failed: %v", err)
			}

			if result.HasMainBlock != tt.hasMain {
				t.Errorf("HasMainBlock = %v, want %v", result.HasMainBlock, tt.hasMain)
			}
		})
	}
}

func TestParseFileFromImportStar(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "star.py")

	content := `from module import *
from another import a, b
`
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parser := NewParser()
	result, err := parser.ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// "module" should have no imports (star is filtered)
	if names, ok := result.FromImports["module"]; ok && len(names) > 0 {
		t.Errorf("expected no names for star import, got %v", names)
	}

	// "another" should have a and b
	if names, ok := result.FromImports["another"]; !ok || len(names) != 2 {
		t.Errorf("expected 2 names for 'another', got %v", names)
	}
}

func TestParseFileParenthesizedImportsSameLine(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "paren.py")

	// Single-line parenthesized import (fully supported)
	content := `from collections import (defaultdict, OrderedDict)
`
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parser := NewParser()
	result, err := parser.ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Should have collections with defaultdict and OrderedDict
	if _, ok := result.FromImports["collections"]; !ok {
		t.Error("expected 'collections' in FromImports")
	}
	if len(result.FromImports["collections"]) != 2 {
		t.Errorf("expected 2 imports from collections, got %d", len(result.FromImports["collections"]))
	}
}

func TestParseFileParenthesizedImportsMultiLine(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "paren_multi.py")

	// Multi-line parenthesized import - only first line is captured with basic parser
	content := `from collections import (
    defaultdict,
    OrderedDict,
    Counter,
)
import os
`
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parser := NewParser()
	result, err := parser.ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Multi-line parenthesized imports are not fully supported by the simple line-based parser
	// The opening paren line doesn't match the regex pattern because it has no names after import (
	// This test documents the current behavior
	// os import should still work
	foundOS := false
	for _, imp := range result.Imports {
		if imp == "os" {
			foundOS = true
		}
	}
	if !foundOS {
		t.Error("expected 'os' import to be found")
	}
}

func TestParseFileBackslashContinuation(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "backslash.py")

	content := `from collections import defaultdict, \
    OrderedDict
import os
`
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parser := NewParser()
	result, err := parser.ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Should have collections
	if _, ok := result.FromImports["collections"]; !ok {
		t.Error("expected 'collections' in FromImports")
	}
	// Should have os
	foundOS := false
	for _, imp := range result.Imports {
		if imp == "os" {
			foundOS = true
		}
	}
	if !foundOS {
		t.Error("expected 'os' import not found")
	}
}

func TestParseFileImportWithAlias(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "alias.py")

	content := `import numpy as np
import pandas as pd
from collections import defaultdict as dd
`
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parser := NewParser()
	result, err := parser.ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Should have numpy and pandas (not np and pd)
	foundNumpy := false
	foundPandas := false
	for _, imp := range result.Imports {
		if imp == "numpy" {
			foundNumpy = true
		}
		if imp == "pandas" {
			foundPandas = true
		}
		if imp == "np" || imp == "pd" {
			t.Errorf("should have module name not alias, got %q", imp)
		}
	}
	if !foundNumpy {
		t.Error("expected 'numpy' import not found")
	}
	if !foundPandas {
		t.Error("expected 'pandas' import not found")
	}

	// Should have defaultdict not dd
	if names, ok := result.FromImports["collections"]; ok {
		for _, name := range names {
			if name == "dd" {
				t.Error("should have 'defaultdict' not alias 'dd'")
			}
		}
	}
}

func TestParseFileMixedTabsAndSpaces(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "mixed.py")

	// Mix tabs and spaces in indentation
	content := "import os\n\timport sys\n    import json\n"
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parser := NewParser()
	result, err := parser.ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Should capture all three imports
	expectedImports := map[string]bool{"os": true, "sys": true, "json": true}
	for _, imp := range result.Imports {
		delete(expectedImports, imp)
	}
	if len(expectedImports) > 0 {
		t.Errorf("missing expected imports: %v", expectedImports)
	}
}

func TestParseFileDottedImports(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "dotted.py")

	content := `import os.path
import urllib.parse
import xml.etree.ElementTree
from collections.abc import Mapping
`
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parser := NewParser()
	result, err := parser.ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Should have top-level modules only: os, urllib, xml
	expectedImports := map[string]bool{"os": true, "urllib": true, "xml": true}
	for _, imp := range result.Imports {
		if !expectedImports[imp] {
			t.Errorf("unexpected import %q (should be top-level only)", imp)
		}
		delete(expectedImports, imp)
	}
	if len(expectedImports) > 0 {
		t.Errorf("missing expected imports: %v", expectedImports)
	}

	// FromImports should have collections (top-level)
	if _, ok := result.FromImports["collections"]; !ok {
		t.Error("expected 'collections' in FromImports")
	}
}

func TestParseFileNonexistent(t *testing.T) {
	parser := NewParser()
	_, err := parser.ParseFile("/nonexistent/path/to/file.py")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestParseFilesMultiple(t *testing.T) {
	tmpDir := t.TempDir()

	// Create first file
	file1 := filepath.Join(tmpDir, "file1.py")
	content1 := `import os`
	if err := os.WriteFile(file1, []byte(content1), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Create second file
	file2 := filepath.Join(tmpDir, "file2.py")
	content2 := `import json`
	if err := os.WriteFile(file2, []byte(content2), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parser := NewParser()
	results, err := parser.ParseFiles([]string{file1, file2})
	if err != nil {
		t.Fatalf("ParseFiles failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestParseFilesWithErrors(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid file
	validFile := filepath.Join(tmpDir, "valid.py")
	if err := os.WriteFile(validFile, []byte("import os"), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Include nonexistent file - should be skipped silently
	parser := NewParser()
	results, err := parser.ParseFiles([]string{validFile, "/nonexistent.py"})
	if err != nil {
		t.Fatalf("ParseFiles should not return error: %v", err)
	}

	// Should only have one result (the valid file)
	if len(results) != 1 {
		t.Errorf("expected 1 result (invalid file skipped), got %d", len(results))
	}
}

// ============================================================================
// isTestFile() Extended Tests
// ============================================================================

func TestIsTestFileExtended(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		// Positive cases
		{"test_example.py", true},
		{"example_test.py", true},
		{"tests/test_foo.py", true},
		{"src/tests/bar.py", true},
		{"test_foo_bar.py", true},
		{"foo_bar_test.py", true},
		{"TEST_EXAMPLE.PY", true}, // case insensitive
		{"EXAMPLE_TEST.PY", true}, // case insensitive
		{"/path/to/tests/module.py", true},
		{"project/tests/subdir/file.py", true},

		// Negative cases - tests/foo.py without leading / doesn't match /tests/
		{"tests/foo.py", false}, // doesn't match /tests/ pattern (needs leading /)
		{"example.py", false},
		{"my_module.py", false},
		{"testing.py", false}, // contains "test" but not a test file pattern
		{"attestation.py", false},
		{"contest.py", false},
		{"latest.py", false},
		{"mytest.py", false}, // not test_ prefix, not _test.py suffix
		{"test.py", false},   // just "test" not "test_"
		{"unittest.py", false},
		{"src/main.py", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isTestFile(tt.path)
			if result != tt.expected {
				t.Errorf("isTestFile(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

// ============================================================================
// getTopLevelModule() Extended Tests
// ============================================================================

func TestGetTopLevelModuleExtended(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Simple module names
		{"os", "os"},
		{"sys", "sys"},
		{"json", "json"},

		// Dotted module names
		{"os.path", "os"},
		{"collections.abc", "collections"},
		{"urllib.parse", "urllib"},

		// Deeply nested modules
		{"xml.etree.ElementTree", "xml"},
		{"email.mime.multipart", "email"},
		{"a.b.c.d.e.f", "a"},

		// Edge cases
		{"single", "single"},
		{"a.b", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := getTopLevelModule(tt.input)
			if result != tt.expected {
				t.Errorf("getTopLevelModule(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// ============================================================================
// parseImportNames() Extended Tests
// ============================================================================

func TestParseImportNamesExtended(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		// Single import
		{"single", "foo", []string{"foo"}},
		{"single_with_spaces", "  foo  ", []string{"foo"}},

		// Multiple imports
		{"multiple", "foo, bar, baz", []string{"foo", "bar", "baz"}},
		{"multiple_no_spaces", "foo,bar,baz", []string{"foo", "bar", "baz"}},
		{"multiple_extra_spaces", "foo ,  bar ,  baz", []string{"foo", "bar", "baz"}},

		// Imports with aliases
		{"single_alias", "foo as f", []string{"foo"}},
		{"multiple_aliases", "foo as f, bar as b", []string{"foo", "bar"}},
		{"mixed_aliases", "foo, bar as b, baz", []string{"foo", "bar", "baz"}},

		// Parenthesized imports
		{"parenthesized", "(foo, bar)", []string{"foo", "bar"}},
		{"parenthesized_single", "(foo)", []string{"foo"}},
		{"parenthesized_with_alias", "(foo as f, bar)", []string{"foo", "bar"}},

		// Star import (should be filtered)
		{"star", "*", []string{}},
		{"star_with_spaces", "  *  ", []string{}},

		// Backslash continuation
		{"backslash", "foo, bar\\", []string{"foo", "bar"}},

		// Empty and edge cases
		{"empty", "", []string{}},
		{"whitespace_only", "   ", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseImportNames(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("parseImportNames(%q) = %v (len=%d), want %v (len=%d)",
					tt.input, result, len(result), tt.expected, len(tt.expected))
				return
			}
			for i, exp := range tt.expected {
				if result[i] != exp {
					t.Errorf("parseImportNames(%q)[%d] = %q, want %q", tt.input, i, result[i], exp)
				}
			}
		})
	}
}

// ============================================================================
// GetAllImports() Extended Tests
// ============================================================================

func TestGetAllImportsDeduplication(t *testing.T) {
	result := &ParseResult{
		Imports: []string{"os", "sys", "os", "json", "sys"}, // duplicates
		FromImports: map[string][]string{
			"os":          {"path"},    // os is also in Imports
			"collections": {"defaultdict"},
		},
	}

	allImports := result.GetAllImports()

	// Count occurrences of each import
	counts := make(map[string]int)
	for _, imp := range allImports {
		counts[imp]++
	}

	// Each import should appear exactly once
	for imp, count := range counts {
		if count != 1 {
			t.Errorf("import %q appears %d times, should be 1", imp, count)
		}
	}

	// Check all expected imports are present
	expected := []string{"os", "sys", "json", "collections"}
	for _, exp := range expected {
		if counts[exp] == 0 {
			t.Errorf("expected import %q not found", exp)
		}
	}
}

func TestGetAllImportsEmpty(t *testing.T) {
	result := &ParseResult{
		Imports:     []string{},
		FromImports: make(map[string][]string),
	}

	allImports := result.GetAllImports()

	if len(allImports) != 0 {
		t.Errorf("expected empty result, got %v", allImports)
	}
}

func TestGetAllImportsOnlyImports(t *testing.T) {
	result := &ParseResult{
		Imports:     []string{"os", "sys"},
		FromImports: make(map[string][]string),
	}

	allImports := result.GetAllImports()

	if len(allImports) != 2 {
		t.Errorf("expected 2 imports, got %d", len(allImports))
	}
}

func TestGetAllImportsOnlyFromImports(t *testing.T) {
	result := &ParseResult{
		Imports: []string{},
		FromImports: map[string][]string{
			"collections": {"defaultdict"},
			"typing":      {"List"},
		},
	}

	allImports := result.GetAllImports()

	if len(allImports) != 2 {
		t.Errorf("expected 2 imports, got %d", len(allImports))
	}
}

// ============================================================================
// Complex Integration Tests
// ============================================================================

func TestParseFileComplexPythonFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "complex.py")

	content := `#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Module docstring.
This is a complex Python file for testing.
"""

import os
import sys
from collections import defaultdict, OrderedDict
from typing import List, Dict, Optional
import json as js
import xml.etree.ElementTree as ET

# Regular comment
from pathlib import Path

class MyClass:
    """Class docstring."""
    pass

def my_function():
    """Function docstring."""
    import functools  # This should be captured
    return None

if __name__ == "__main__":
    print("Hello, World!")
`
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parser := NewParser()
	result, err := parser.ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Check imports
	expectedImports := map[string]bool{
		"os":        true,
		"sys":       true,
		"json":      true,
		"xml":       true,
		"functools": true,
	}
	for _, imp := range result.Imports {
		delete(expectedImports, imp)
	}
	if len(expectedImports) > 0 {
		t.Errorf("missing expected imports: %v", expectedImports)
	}

	// Check from imports
	expectedFromImports := []string{"collections", "typing", "pathlib"}
	for _, exp := range expectedFromImports {
		if _, ok := result.FromImports[exp]; !ok {
			t.Errorf("expected %q in FromImports", exp)
		}
	}

	// Check main block
	if !result.HasMainBlock {
		t.Error("expected HasMainBlock to be true")
	}
}

func TestParseFileWithUnusualImportPatterns(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "unusual.py")

	content := `# Various import patterns

# Standard import
import os

# Import with very long alias
import numpy as this_is_a_very_long_alias_name

# From import with parentheses on same line
from typing import (List, Dict)

# From import with trailing comma
from collections import defaultdict,

# Import of package starting with underscore
import _thread
`
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	parser := NewParser()
	result, err := parser.ParseFile(testFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Should have os, numpy, _thread
	expectedImports := map[string]bool{"os": true, "numpy": true, "_thread": true}
	for _, imp := range result.Imports {
		delete(expectedImports, imp)
	}
	if len(expectedImports) > 0 {
		t.Errorf("missing expected imports: %v", expectedImports)
	}
}

func TestNewParser(t *testing.T) {
	parser := NewParser()

	if parser == nil {
		t.Fatal("NewParser() returned nil")
	}
	if parser.importRegex == nil {
		t.Error("importRegex is nil")
	}
	if parser.fromImportRegex == nil {
		t.Error("fromImportRegex is nil")
	}
	if parser.mainBlockRegex == nil {
		t.Error("mainBlockRegex is nil")
	}
}
