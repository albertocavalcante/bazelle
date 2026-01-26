package kotlin

import (
	"context"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/albertocavalcante/bazelle/pkg/treesitter"
)

var ctx = context.Background()

func TestHeuristicBackend_ParseContent(t *testing.T) {
	cfg := DefaultBackendConfig()
	backend := NewHeuristicBackend(cfg)
	defer backend.Close()

	content := `package com.example.myapp

import kotlin.test.Test
import org.junit.jupiter.api.BeforeEach

class MyTest {
    @Test
    fun testSomething() {}
}
`

	result, err := backend.ParseContent(ctx, content, "Test.kt")
	if err != nil || result == nil {
		t.Fatalf("ParseContent failed: %v", err)
	}

	if result.Package != "com.example.myapp" {
		t.Errorf("Package: expected 'com.example.myapp', got '%s'", result.Package)
	}

	expectedImports := []string{
		"kotlin.test.Test",
		"org.junit.jupiter.api.BeforeEach",
	}
	if !reflect.DeepEqual(result.Imports, expectedImports) {
		t.Errorf("Imports: expected %v, got %v", expectedImports, result.Imports)
	}
}

func TestTreeSitterBackend_ParseContent(t *testing.T) {
	backends := treesitter.AvailableBackends()
	if len(backends) == 0 {
		t.Skip("No tree-sitter backends available")
	}

	cfg := DefaultBackendConfig()
	backend, err := NewTreeSitterBackend(cfg)
	if err != nil || backend == nil {
		t.Fatalf("Failed to create TreeSitterBackend: %v", err)
	}
	defer backend.Close()

	content := `package com.example.myapp

import kotlin.test.Test
import org.junit.jupiter.api.BeforeEach

class MyTest {
    @Test
    fun testSomething() {}
}
`

	result, err := backend.ParseContent(ctx, content, "Test.kt")
	if err != nil || result == nil {
		t.Fatalf("ParseContent failed: %v", err)
	}

	if result.Package != "com.example.myapp" {
		t.Errorf("Package: expected 'com.example.myapp', got '%s'", result.Package)
	}

	// Check that imports are present (may not be in exact order)
	importSet := make(map[string]bool)
	for _, imp := range result.Imports {
		importSet[imp] = true
	}

	expectedImports := []string{
		"kotlin.test.Test",
		"org.junit.jupiter.api.BeforeEach",
	}
	for _, expected := range expectedImports {
		if !importSet[expected] {
			t.Errorf("Expected import '%s' not found in %v", expected, result.Imports)
		}
	}
}

func TestTreeSitterBackend_StarImports(t *testing.T) {
	backends := treesitter.AvailableBackends()
	if len(backends) == 0 {
		t.Skip("No tree-sitter backends available")
	}

	cfg := DefaultBackendConfig()
	backend, err := NewTreeSitterBackend(cfg)
	if err != nil || backend == nil {
		t.Fatalf("Failed to create TreeSitterBackend: %v", err)
	}
	defer backend.Close()

	content := `package com.example

import kotlin.collections.*
import org.junit.jupiter.api.*

class Foo
`

	result, err := backend.ParseContent(ctx, content, "Foo.kt")
	if err != nil || result == nil {
		t.Fatalf("ParseContent failed: %v", err)
	}

	slices.Sort(result.StarImports)
	expected := []string{"kotlin.collections", "org.junit.jupiter.api"}
	if !reflect.DeepEqual(result.StarImports, expected) {
		t.Errorf("StarImports: expected %v, got %v", expected, result.StarImports)
	}
}

func TestTreeSitterBackend_ImportAlias(t *testing.T) {
	backends := treesitter.AvailableBackends()
	if len(backends) == 0 {
		t.Skip("No tree-sitter backends available")
	}

	cfg := DefaultBackendConfig()
	backend, err := NewTreeSitterBackend(cfg)
	if err != nil || backend == nil {
		t.Fatalf("Failed to create TreeSitterBackend: %v", err)
	}
	defer backend.Close()

	content := `package com.example

import com.example.LongClassName as Short
import org.something.Else as Other

class Foo
`

	result, err := backend.ParseContent(ctx, content, "Foo.kt")
	if err != nil || result == nil {
		t.Fatalf("ParseContent failed: %v", err)
	}

	// Check aliases
	if result.ImportAliases["Short"] != "com.example.LongClassName" {
		t.Errorf("Expected alias 'Short' -> 'com.example.LongClassName', got %v", result.ImportAliases)
	}
	if result.ImportAliases["Other"] != "org.something.Else" {
		t.Errorf("Expected alias 'Other' -> 'org.something.Else', got %v", result.ImportAliases)
	}
}

func TestHybridBackend_ComparesResults(t *testing.T) {
	backends := treesitter.AvailableBackends()
	if len(backends) == 0 {
		t.Skip("No tree-sitter backends available")
	}

	cfg := DefaultBackendConfig()
	cfg.HybridLogDiffs = false // Suppress log output in tests
	backend, err := NewHybridBackend(cfg)
	if err != nil || backend == nil {
		t.Fatalf("Failed to create HybridBackend: %v", err)
	}
	defer backend.Close()

	content := `package com.example.myapp

import kotlin.test.Test

class MyTest
`

	result, err := backend.ParseContent(ctx, content, "Test.kt")
	if err != nil || result == nil {
		t.Fatalf("ParseContent failed: %v", err)
	}

	// Hybrid should return a valid result
	if result.Package != "com.example.myapp" {
		t.Errorf("Package: expected 'com.example.myapp', got '%s'", result.Package)
	}
}

func TestHybridBackend_UsesPrimary(t *testing.T) {
	backends := treesitter.AvailableBackends()
	if len(backends) == 0 {
		t.Skip("No tree-sitter backends available")
	}

	tests := []struct {
		name    string
		primary ParserBackendType
	}{
		{"heuristic primary", BackendHeuristic},
		{"treesitter primary", BackendTreeSitter},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := DefaultBackendConfig()
			cfg.HybridPrimary = tc.primary
			cfg.HybridLogDiffs = false

			backend, err := NewHybridBackend(cfg)
			if err != nil || backend == nil {
				t.Fatalf("Failed to create HybridBackend: %v", err)
			}
			defer backend.Close()

			content := `package com.example

import kotlin.test.Test

class Foo
`
			result, err := backend.ParseContent(ctx, content, "Test.kt")
			if err != nil || result == nil {
				t.Fatalf("ParseContent failed: %v", err)
			}

			if result.Package != "com.example" {
				t.Errorf("Package: expected 'com.example', got '%s'", result.Package)
			}
		})
	}
}

func TestNewParserBackend_InvalidType(t *testing.T) {
	cfg := DefaultBackendConfig()
	_, err := NewParserBackend("invalid", cfg)
	if err == nil {
		t.Error("Expected error for invalid backend type")
	}

	// Check it's the right error type
	expected := ErrBackendNotSupported{Backend: "invalid", Reason: "unknown type"}
	if !reflect.DeepEqual(err, expected) {
		t.Errorf("Expected ErrBackendNotSupported{invalid, unknown type}, got %T: %v", err, err)
	}
}

func TestBackendConfig_Defaults(t *testing.T) {
	cfg := DefaultBackendConfig()

	if !cfg.EnableFQNScanning {
		t.Error("EnableFQNScanning should be true by default")
	}
	if cfg.HybridPrimary != BackendHeuristic {
		t.Errorf("HybridPrimary should be 'heuristic' by default, got '%s'", cfg.HybridPrimary)
	}
	if !cfg.HybridLogDiffs {
		t.Error("HybridLogDiffs should be true by default")
	}
}

// TestBackendConsistency verifies that heuristic and tree-sitter produce
// consistent results for well-formed Kotlin code.
func TestBackendConsistency(t *testing.T) {
	backends := treesitter.AvailableBackends()
	if len(backends) == 0 {
		t.Skip("No tree-sitter backends available")
	}

	cfg := DefaultBackendConfig()
	cfg.EnableFQNScanning = false // Disable FQN for consistency test

	heuristic := NewHeuristicBackend(cfg)
	defer heuristic.Close()

	ts, err := NewTreeSitterBackend(cfg)
	if err != nil || ts == nil {
		t.Fatalf("Failed to create TreeSitterBackend: %v", err)
	}
	defer ts.Close()

	testCases := []struct {
		name    string
		content string
	}{
		{
			name: "simple package and imports",
			content: `package com.example.myapp

import kotlin.test.Test
import org.junit.jupiter.api.BeforeEach

class MyTest
`,
		},
		{
			name: "star imports",
			content: `package com.example

import kotlin.collections.*
import org.junit.jupiter.api.*

class Foo
`,
		},
		{
			name: "aliased imports",
			content: `package com.example

import com.example.LongName as Short
import org.something.Else as Other

class Foo
`,
		},
		{
			name: "no imports",
			content: `package com.example

class Empty
`,
		},
		{
			name: "no package",
			content: `import kotlin.test.Test

class NoPackage
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hResult, hErr := heuristic.ParseContent(ctx, tc.content, "Test.kt")
			tsResult, tsErr := ts.ParseContent(ctx, tc.content, "Test.kt")

			if hErr != nil || hResult == nil {
				t.Fatalf("Heuristic parsing failed: %v", hErr)
			}
			if tsErr != nil || tsResult == nil {
				t.Fatalf("TreeSitter parsing failed: %v", tsErr)
			}

			// Compare packages
			if hResult.Package != tsResult.Package {
				t.Errorf("Package mismatch: heuristic=%q, treesitter=%q",
					hResult.Package, tsResult.Package)
			}

			// Compare imports (as sets, order may differ)
			hImports := toStringSet(hResult.Imports)
			tsImports := toStringSet(tsResult.Imports)
			if !reflect.DeepEqual(hImports, tsImports) {
				t.Errorf("Imports mismatch:\n  heuristic:  %v\n  treesitter: %v",
					hResult.Imports, tsResult.Imports)
			}

			// Compare star imports
			hStars := toStringSet(hResult.StarImports)
			tsStars := toStringSet(tsResult.StarImports)
			if !reflect.DeepEqual(hStars, tsStars) {
				t.Errorf("StarImports mismatch:\n  heuristic:  %v\n  treesitter: %v",
					hResult.StarImports, tsResult.StarImports)
			}

			// Compare aliases
			if !reflect.DeepEqual(hResult.ImportAliases, tsResult.ImportAliases) {
				t.Errorf("ImportAliases mismatch:\n  heuristic:  %v\n  treesitter: %v",
					hResult.ImportAliases, tsResult.ImportAliases)
			}
		})
	}
}

func TestResultDiff_HasDifferences(t *testing.T) {
	tests := []struct {
		name     string
		diff     ResultDiff
		expected bool
	}{
		{"empty diff", ResultDiff{}, false},
		{"package diff", ResultDiff{PackageDiff: &[2]string{"a", "b"}}, true},
		{"imports diff", ResultDiff{OnlyInHeuristic: []string{"foo"}}, true},
		{"star diff", ResultDiff{StarOnlyTreeSit: []string{"bar"}}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.diff.HasDifferences(); got != tc.expected {
				t.Errorf("HasDifferences() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestResultDiff_String(t *testing.T) {
	diff := ResultDiff{
		PackageDiff:      &[2]string{"pkg1", "pkg2"},
		OnlyInHeuristic:  []string{"import1"},
		OnlyInTreeSitter: []string{"import2"},
	}

	s := diff.String()
	if s == "" {
		t.Error("Expected non-empty string")
	}
	if !strings.Contains(s, "pkg1") || !strings.Contains(s, "pkg2") {
		t.Error("Expected package names in diff string")
	}
	if !strings.Contains(s, "import1") || !strings.Contains(s, "import2") {
		t.Error("Expected import names in diff string")
	}
}

func TestCompareResults(t *testing.T) {
	h := &ParseResult{
		Package:     "com.example",
		Imports:     []string{"a", "b", "c"},
		StarImports: []string{"pkg1"},
	}
	ts := &ParseResult{
		Package:     "com.example",
		Imports:     []string{"b", "c", "d"},
		StarImports: []string{"pkg1", "pkg2"},
	}

	diff := compareResults(h, ts)

	if diff.PackageDiff != nil {
		t.Error("Package should match")
	}
	if !reflect.DeepEqual(diff.OnlyInHeuristic, []string{"a"}) {
		t.Errorf("OnlyInHeuristic: expected [a], got %v", diff.OnlyInHeuristic)
	}
	if !reflect.DeepEqual(diff.OnlyInTreeSitter, []string{"d"}) {
		t.Errorf("OnlyInTreeSitter: expected [d], got %v", diff.OnlyInTreeSitter)
	}
	if len(diff.StarOnlyHeuristic) != 0 {
		t.Errorf("StarOnlyHeuristic: expected empty, got %v", diff.StarOnlyHeuristic)
	}
	if !reflect.DeepEqual(diff.StarOnlyTreeSit, []string{"pkg2"}) {
		t.Errorf("StarOnlyTreeSit: expected [pkg2], got %v", diff.StarOnlyTreeSit)
	}
}

func TestErrBackendNotSupported_Error(t *testing.T) {
	err := ErrBackendNotSupported{Backend: "test", Reason: "some reason"}
	if !strings.Contains(err.Error(), "test") || !strings.Contains(err.Error(), "some reason") {
		t.Errorf("Error message should contain backend and reason: %s", err.Error())
	}

	err2 := ErrBackendNotSupported{Backend: "test"}
	if !strings.Contains(err2.Error(), "test") {
		t.Errorf("Error message should contain backend: %s", err2.Error())
	}
}

func TestErrLanguageNotSupported_Error(t *testing.T) {
	err := ErrLanguageNotSupported{Backend: "wazero"}
	if !strings.Contains(err.Error(), "wazero") || !strings.Contains(err.Error(), "Kotlin") {
		t.Errorf("Error message should contain backend and Kotlin: %s", err.Error())
	}
}
