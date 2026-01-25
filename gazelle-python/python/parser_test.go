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
