package kotlin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParser_ParseFile(t *testing.T) {
	// Create a temporary Kotlin file
	tmpDir := t.TempDir()
	ktFile := filepath.Join(tmpDir, "Test.kt")

	content := `package com.example.myapp

import kotlin.test.Test
import kotlin.test.assertEquals
import org.junit.jupiter.api.BeforeEach

class MyTest {
    @Test
    fun testSomething() {
        assertEquals(1, 1)
    }
}
`
	if err := os.WriteFile(ktFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	parser := NewParser()
	result, err := parser.ParseFile(ktFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Check package
	if result.Package != "com.example.myapp" {
		t.Errorf("Expected package 'com.example.myapp', got '%s'", result.Package)
	}

	// Check imports
	expectedImports := []string{
		"kotlin.test.Test",
		"kotlin.test.assertEquals",
		"org.junit.jupiter.api.BeforeEach",
	}
	if len(result.Imports) != len(expectedImports) {
		t.Errorf("Expected %d imports, got %d", len(expectedImports), len(result.Imports))
	}
	for i, expected := range expectedImports {
		if i < len(result.Imports) && result.Imports[i] != expected {
			t.Errorf("Import %d: expected '%s', got '%s'", i, expected, result.Imports[i])
		}
	}
}

func TestParser_ParseFile_WithComments(t *testing.T) {
	tmpDir := t.TempDir()
	ktFile := filepath.Join(tmpDir, "Commented.kt")

	content := `// Copyright 2025
/*
 * Multi-line comment
 * import fake.Import
 */
package com.example.commented

// Single line comment
import kotlin.collections.List

class Commented
`
	if err := os.WriteFile(ktFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	parser := NewParser()
	result, err := parser.ParseFile(ktFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if result.Package != "com.example.commented" {
		t.Errorf("Expected package 'com.example.commented', got '%s'", result.Package)
	}

	// Should only have one import (not the fake one in comment)
	if len(result.Imports) != 1 {
		t.Errorf("Expected 1 import, got %d: %v", len(result.Imports), result.Imports)
	}
	if len(result.Imports) > 0 && result.Imports[0] != "kotlin.collections.List" {
		t.Errorf("Expected import 'kotlin.collections.List', got '%s'", result.Imports[0])
	}
}

func TestGetPackages(t *testing.T) {
	results := []*ParseResult{
		{Package: "com.example.a"},
		{Package: "com.example.b"},
		{Package: "com.example.a"}, // Duplicate
	}

	packages := GetPackages(results)
	if len(packages) != 2 {
		t.Errorf("Expected 2 unique packages, got %d", len(packages))
	}
}
