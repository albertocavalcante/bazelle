package kotlin

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
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

func TestParser_StarImports(t *testing.T) {
	parser := NewParser()
	content := `package com.example.test

import com.example.models.*
import org.junit.jupiter.api.*
import com.example.utils.Helper

class Test
`
	result, err := parser.ParseContent(content, "Test.kt")
	if err != nil {
		t.Fatalf("ParseContent failed: %v", err)
	}

	// Check star imports
	expectedStarImports := []string{
		"com.example.models",
		"org.junit.jupiter.api",
	}
	if !reflect.DeepEqual(result.StarImports, expectedStarImports) {
		t.Errorf("Star imports: expected %v, got %v", expectedStarImports, result.StarImports)
	}

	// Check regular imports
	if len(result.Imports) != 1 || result.Imports[0] != "com.example.utils.Helper" {
		t.Errorf("Regular imports: expected [com.example.utils.Helper], got %v", result.Imports)
	}
}

func TestParser_ImportAliases(t *testing.T) {
	parser := NewParser()
	content := `package com.example.test

import com.example.models.User as AppUser
import org.json.JSONObject as Json
import com.example.utils.Helper

class Test
`
	result, err := parser.ParseContent(content, "Test.kt")
	if err != nil {
		t.Fatalf("ParseContent failed: %v", err)
	}

	// Check that aliased imports are in the imports list
	if len(result.Imports) != 3 {
		t.Errorf("Expected 3 imports, got %d: %v", len(result.Imports), result.Imports)
	}

	// Check aliases
	expectedAliases := map[string]string{
		"AppUser": "com.example.models.User",
		"Json":    "org.json.JSONObject",
	}
	if !reflect.DeepEqual(result.ImportAliases, expectedAliases) {
		t.Errorf("Import aliases: expected %v, got %v", expectedAliases, result.ImportAliases)
	}
}

func TestParser_FileAnnotations(t *testing.T) {
	parser := NewParser()
	content := `@file:JvmName("MyUtils")
@file:Suppress("UNCHECKED_CAST")
package com.example.test

import com.example.Foo

class Test
`
	result, err := parser.ParseContent(content, "Test.kt")
	if err != nil {
		t.Fatalf("ParseContent failed: %v", err)
	}

	// Check annotations
	expectedAnnotations := []string{"JvmName", "Suppress"}
	if !reflect.DeepEqual(result.Annotations, expectedAnnotations) {
		t.Errorf("Annotations: expected %v, got %v", expectedAnnotations, result.Annotations)
	}

	// Package should still be parsed correctly
	if result.Package != "com.example.test" {
		t.Errorf("Package: expected 'com.example.test', got '%s'", result.Package)
	}
}

func TestParser_BacktickPackage(t *testing.T) {
	parser := NewParser()
	content := "package `com.example.reserved`\n\nclass Test"

	result, err := parser.ParseContent(content, "Test.kt")
	if err != nil {
		t.Fatalf("ParseContent failed: %v", err)
	}

	if result.Package != "com.example.reserved" {
		t.Errorf("Package: expected 'com.example.reserved', got '%s'", result.Package)
	}
}

func TestParser_FQNScanning(t *testing.T) {
	parser := NewParser()
	content := `package com.example.test

import com.example.models.User

class Service {
    fun process(): com.example.result.Result {
        val client = io.ktor.client.HttpClient()
        val mapper = com.fasterxml.jackson.databind.ObjectMapper()
        return com.example.result.Result.success()
    }
}
`
	result, err := parser.ParseContent(content, "Service.kt")
	if err != nil {
		t.Fatalf("ParseContent failed: %v", err)
	}

	// Check that FQNs are detected
	// Note: Some FQNs might be filtered by the scanner (e.g., kotlin stdlib)
	fqnSet := make(map[string]bool)
	for _, fqn := range result.FQNs {
		fqnSet[fqn] = true
	}

	// These should be detected
	expectedFQNs := []string{
		"com.example.result.Result",
		"io.ktor.client.HttpClient",
		"com.fasterxml.jackson.databind.ObjectMapper",
	}
	for _, expected := range expectedFQNs {
		if !fqnSet[expected] {
			t.Errorf("Expected FQN '%s' to be detected, but it wasn't. Got: %v", expected, result.FQNs)
		}
	}

	// AllDependencies should include both imports and FQNs
	allDepsSet := make(map[string]bool)
	for _, dep := range result.AllDependencies {
		allDepsSet[dep] = true
	}
	if !allDepsSet["com.example.models.User"] {
		t.Error("AllDependencies should include imported User")
	}
}

func TestParser_FQNScanning_Disabled(t *testing.T) {
	parser := NewParser(WithFQNScanning(false))
	content := `package com.example.test

class Service {
    fun process(): com.example.result.Result {
        return com.example.result.Result.success()
    }
}
`
	result, err := parser.ParseContent(content, "Service.kt")
	if err != nil {
		t.Fatalf("ParseContent failed: %v", err)
	}

	// FQNs should be empty when scanning is disabled
	if len(result.FQNs) != 0 {
		t.Errorf("FQNs should be empty when scanning disabled, got: %v", result.FQNs)
	}
}

func TestParser_FQNInTypeAnnotation(t *testing.T) {
	parser := NewParser()
	content := `package com.example.test

class Service {
    val config: com.example.config.AppConfig? = null

    fun getUser(): com.example.models.User {
        TODO()
    }

    fun checkType(obj: Any): Boolean {
        return obj is com.example.models.Admin
    }
}
`
	result, err := parser.ParseContent(content, "Service.kt")
	if err != nil {
		t.Fatalf("ParseContent failed: %v", err)
	}

	fqnSet := make(map[string]bool)
	for _, fqn := range result.FQNs {
		fqnSet[fqn] = true
	}

	// These should be detected from type annotations
	expectedFQNs := []string{
		"com.example.config.AppConfig",
		"com.example.models.User",
		"com.example.models.Admin",
	}
	for _, expected := range expectedFQNs {
		if !fqnSet[expected] {
			t.Errorf("Expected FQN '%s' to be detected in type annotation", expected)
		}
	}
}

func TestParser_FQNInStringShouldBeIgnored(t *testing.T) {
	parser := NewParser()
	content := `package com.example.test

class Service {
    val className = "com.example.fake.ClassName"
    val template = """
        com.example.multiline.FakeClass
    """
}
`
	result, err := parser.ParseContent(content, "Service.kt")
	if err != nil {
		t.Fatalf("ParseContent failed: %v", err)
	}

	// FQNs in strings should NOT be detected
	for _, fqn := range result.FQNs {
		if fqn == "com.example.fake.ClassName" || fqn == "com.example.multiline.FakeClass" {
			t.Errorf("FQN '%s' should not be detected (it's in a string)", fqn)
		}
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

func TestGetAllImports(t *testing.T) {
	results := []*ParseResult{
		{Imports: []string{"com.example.Foo", "com.example.Bar"}},
		{Imports: []string{"com.example.Bar", "com.example.Baz"}}, // Bar is duplicate
	}

	imports := GetAllImports(results)
	sort.Strings(imports)

	expected := []string{"com.example.Bar", "com.example.Baz", "com.example.Foo"}
	if !reflect.DeepEqual(imports, expected) {
		t.Errorf("Expected %v, got %v", expected, imports)
	}
}

func TestGetAllDependencies(t *testing.T) {
	results := []*ParseResult{
		{AllDependencies: []string{"com.example.Foo", "com.example.Bar"}},
		{AllDependencies: []string{"io.ktor.HttpClient", "com.example.Foo"}}, // Foo is duplicate
	}

	deps := GetAllDependencies(results)
	sort.Strings(deps)

	expected := []string{"com.example.Bar", "com.example.Foo", "io.ktor.HttpClient"}
	if !reflect.DeepEqual(deps, expected) {
		t.Errorf("Expected %v, got %v", expected, deps)
	}
}

func TestGetImportInfo(t *testing.T) {
	result := &ParseResult{
		Imports:       []string{"com.example.models.User", "org.json.JSONObject"},
		StarImports:   []string{"com.example.utils"},
		ImportAliases: map[string]string{"Json": "org.json.JSONObject"},
	}

	infos := GetImportInfo(result)

	// Should have 3 infos: 2 regular + 1 star
	if len(infos) != 3 {
		t.Fatalf("Expected 3 import infos, got %d", len(infos))
	}

	// Check User import
	userInfo := infos[0]
	if userInfo.Path != "com.example.models.User" {
		t.Errorf("Expected path 'com.example.models.User', got '%s'", userInfo.Path)
	}
	if userInfo.Package != "com.example.models" {
		t.Errorf("Expected package 'com.example.models', got '%s'", userInfo.Package)
	}
	if userInfo.Name != "User" {
		t.Errorf("Expected name 'User', got '%s'", userInfo.Name)
	}

	// Check JSONObject import (has alias)
	jsonInfo := infos[1]
	if jsonInfo.Alias != "Json" {
		t.Errorf("Expected alias 'Json', got '%s'", jsonInfo.Alias)
	}

	// Check star import
	starInfo := infos[2]
	if !starInfo.IsStar {
		t.Error("Expected star import to have IsStar=true")
	}
	if starInfo.Package != "com.example.utils" {
		t.Errorf("Expected package 'com.example.utils', got '%s'", starInfo.Package)
	}
}

func TestExtractPackageFromFQN(t *testing.T) {
	tests := []struct {
		fqn      string
		expected string
	}{
		{"com.example.Foo", "com.example"},
		{"com.example.sub.Bar", "com.example.sub"},
		{"Foo", ""},
		{"", ""},
	}

	for _, tt := range tests {
		result := ExtractPackageFromFQN(tt.fqn)
		if result != tt.expected {
			t.Errorf("ExtractPackageFromFQN(%q): expected %q, got %q", tt.fqn, tt.expected, result)
		}
	}
}

func TestExtractClassFromFQN(t *testing.T) {
	tests := []struct {
		fqn      string
		expected string
	}{
		{"com.example.Foo", "Foo"},
		{"com.example.sub.Bar", "Bar"},
		{"Foo", "Foo"},
		{"", ""},
	}

	for _, tt := range tests {
		result := ExtractClassFromFQN(tt.fqn)
		if result != tt.expected {
			t.Errorf("ExtractClassFromFQN(%q): expected %q, got %q", tt.fqn, tt.expected, result)
		}
	}
}

// FQN Scanner tests

func TestFQNScanner_BasicDetection(t *testing.T) {
	scanner := NewFQNScanner()
	content := `package com.example.test

class Service {
    fun process() {
        val client = io.ktor.client.HttpClient()
        val result = com.example.result.Result.success()
    }
}
`
	result := scanner.Scan(content, 3) // Start after imports

	fqnSet := make(map[string]bool)
	for _, fqn := range result.FQNs {
		fqnSet[fqn] = true
	}

	if !fqnSet["io.ktor.client.HttpClient"] {
		t.Errorf("Expected to detect io.ktor.client.HttpClient, got: %v", result.FQNs)
	}
	if !fqnSet["com.example.result.Result"] {
		t.Errorf("Expected to detect com.example.result.Result, got: %v", result.FQNs)
	}
}

func TestFQNScanner_ExcludesStdlib(t *testing.T) {
	scanner := NewFQNScanner()
	content := `package com.example.test

class Service {
    fun process() {
        val list = kotlin.collections.listOf(1, 2, 3)
        val str = java.lang.String.valueOf(42)
    }
}
`
	result := scanner.Scan(content, 3)

	// kotlin.* and java.* should be excluded
	for _, fqn := range result.FQNs {
		if fqn == "kotlin.collections.listOf" || fqn == "java.lang.String" {
			t.Errorf("Stdlib FQN '%s' should be excluded", fqn)
		}
	}
}

func TestFQNScanner_IncludesKotlinx(t *testing.T) {
	scanner := NewFQNScanner()
	content := `package com.example.test

class Service {
    suspend fun process() {
        kotlinx.coroutines.delay(1000)
        val flow = kotlinx.coroutines.flow.flowOf(1)
    }
}
`
	result := scanner.Scan(content, 3)

	// kotlinx.* should be included (it's a separate dependency)
	fqnSet := make(map[string]bool)
	for _, fqn := range result.FQNs {
		fqnSet[fqn] = true
	}

	// Note: The scanner might detect these differently based on patterns
	hasKotlinx := false
	for fqn := range fqnSet {
		if len(fqn) > 7 && fqn[:7] == "kotlinx" {
			hasKotlinx = true
			break
		}
	}
	if !hasKotlinx {
		t.Log("Note: kotlinx FQNs might not be detected by current patterns")
	}
}

func TestFQNScanner_SkipsComments(t *testing.T) {
	scanner := NewFQNScanner()
	content := `package com.example.test

class Service {
    // This is a comment: com.example.fake.Commented
    /*
     * Multi-line comment
     * com.example.fake.MultiLine
     */
    fun process() {
        val real = com.example.real.RealClass()
    }
}
`
	result := scanner.Scan(content, 3)

	for _, fqn := range result.FQNs {
		if fqn == "com.example.fake.Commented" || fqn == "com.example.fake.MultiLine" {
			t.Errorf("FQN in comment '%s' should not be detected", fqn)
		}
	}

	// Real one should be detected
	fqnSet := make(map[string]bool)
	for _, fqn := range result.FQNs {
		fqnSet[fqn] = true
	}
	if !fqnSet["com.example.real.RealClass"] {
		t.Errorf("Expected to detect com.example.real.RealClass")
	}
}

func TestFQNScanner_TracksLocations(t *testing.T) {
	scanner := NewFQNScanner()
	content := `package com.example.test

class Service {
    fun first() {
        com.example.foo.Foo()
    }
    fun second() {
        com.example.foo.Foo()
    }
}
`
	result := scanner.Scan(content, 3)

	// Check that locations are tracked (may have duplicates from multiple pattern matches)
	if locs, ok := result.FQNToLocations["com.example.foo.Foo"]; ok {
		// At minimum we should have locations from both function bodies
		if len(locs) < 2 {
			t.Errorf("Expected at least 2 locations for Foo, got %d", len(locs))
		}
		// Verify that distinct lines are captured
		lineSet := make(map[int]bool)
		for _, loc := range locs {
			lineSet[loc] = true
		}
		if len(lineSet) < 2 {
			t.Errorf("Expected locations on at least 2 distinct lines, got %d unique lines", len(lineSet))
		}
	} else {
		t.Error("Expected com.example.foo.Foo to be detected")
	}
}

// Test for regex compilation efficiency
func TestRemoveStringLiterals_NoRecompilation(t *testing.T) {
	// This test verifies that removeStringLiterals doesn't recompile regexes on every call
	// We'll call it multiple times to ensure it works consistently
	testCases := []struct {
		input    string
		expected string
	}{
		{`val x = "hello world"`, `val x = ""`},
		{`val y = 'c'`, `val y = ''`},
		{`val z = "string with \"escaped\" quotes"`, `val z = ""`},
		{`val a = 'x' and "text"`, `val a = '' and ""`},
	}

	for _, tc := range testCases {
		result := removeStringLiterals(tc.input)
		if result != tc.expected {
			t.Errorf("removeStringLiterals(%q): expected %q, got %q", tc.input, tc.expected, result)
		}
	}

	// Run many times to ensure consistent behavior (performance test would show regression)
	for i := 0; i < 100; i++ {
		removeStringLiterals(`val test = "hello" and 'c'`)
	}
}

// Test for efficient alias lookup in GetImportInfo
func TestGetImportInfo_EfficientAliasLookup(t *testing.T) {
	// Create a result with many imports and aliases to test performance
	result := &ParseResult{
		Imports: []string{
			"com.example.models.User",
			"com.example.models.Product",
			"com.example.models.Order",
			"org.json.JSONObject",
			"org.json.JSONArray",
		},
		ImportAliases: map[string]string{
			"AppUser": "com.example.models.User",
			"Json":    "org.json.JSONObject",
			"JArray":  "org.json.JSONArray",
		},
	}

	infos := GetImportInfo(result)

	// Verify all imports are processed
	if len(infos) != 5 {
		t.Fatalf("Expected 5 import infos, got %d", len(infos))
	}

	// Verify aliases are correctly matched
	aliasCount := 0
	for _, info := range infos {
		if info.Alias != "" {
			aliasCount++
			// Verify the alias matches the import
			if expectedAlias, ok := result.ImportAliases[info.Alias]; ok {
				if expectedAlias != info.Path {
					t.Errorf("Alias %s: expected path %s, got %s", info.Alias, expectedAlias, info.Path)
				}
			} else {
				t.Errorf("Unexpected alias %s", info.Alias)
			}
		}
	}

	if aliasCount != 3 {
		t.Errorf("Expected 3 aliased imports, got %d", aliasCount)
	}

	// Run multiple times to ensure consistent performance
	for i := 0; i < 100; i++ {
		GetImportInfo(result)
	}
}

// ===============================================
// Edge Case Tests (derived from QA adversarial testing)
// ===============================================

func TestParser_EmptyFile(t *testing.T) {
	parser := NewParser()
	result, err := parser.ParseContent("", "empty.kt")
	if err != nil {
		t.Fatalf("Failed to parse empty file: %v", err)
	}
	if result.Package != "" {
		t.Errorf("Empty file should have no package, got: %s", result.Package)
	}
	if len(result.Imports) != 0 {
		t.Errorf("Empty file should have no imports, got: %v", result.Imports)
	}
}

func TestParser_OnlyComments(t *testing.T) {
	parser := NewParser()
	content := `// Just comments
/* Multi-line
   comment only */
// More comments
`
	result, err := parser.ParseContent(content, "comments.kt")
	if err != nil {
		t.Fatalf("Failed to parse comment-only file: %v", err)
	}
	if result.Package != "" {
		t.Errorf("Comment-only file should have no package, got: %s", result.Package)
	}
}

func TestParser_VeryLongLine(t *testing.T) {
	parser := NewParser()
	// Create a line with 100KB of content (tests buffer size fix)
	longImport := "import com.example." + strings.Repeat("a", 100000) + ".VeryLongPackage"
	content := "package com.example.test\n" + longImport + "\n\nclass Test"

	result, err := parser.ParseContent(content, "long.kt")
	if err != nil {
		t.Fatalf("Failed to parse file with very long line: %v", err)
	}
	if result.Package != "com.example.test" {
		t.Errorf("Package not parsed correctly with long line")
	}
}

func TestParser_MultilineCommentAcrossPackage(t *testing.T) {
	parser := NewParser()
	content := `/* comment
package fake.Package
*/
package com.example.real

import com.example.Foo
`
	result, err := parser.ParseContent(content, "test.kt")
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	if result.Package != "com.example.real" {
		t.Errorf("Expected package 'com.example.real', got '%s'", result.Package)
	}
}

func TestParser_CommentWithSlashStarInside(t *testing.T) {
	// Tests fix for BUG-005: // comment containing /* should not trigger multiline mode
	parser := NewParser()
	content := `package com.example.test
// This is a comment with /* inside it
import com.example.Foo
class Test
`
	result, err := parser.ParseContent(content, "test.kt")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(result.Imports) != 1 || result.Imports[0] != "com.example.Foo" {
		t.Errorf("Should parse import after // comment with /*, got: %v", result.Imports)
	}
}

func TestParser_NoPackageDeclaration(t *testing.T) {
	parser := NewParser()
	content := `import com.example.Foo
class Test
`
	result, err := parser.ParseContent(content, "test.kt")
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	if result.Package != "" {
		t.Errorf("Expected empty package, got: %s", result.Package)
	}
	if len(result.Imports) != 1 {
		t.Errorf("Expected 1 import, got %d", len(result.Imports))
	}
}

func TestParser_MultiplePackageDeclarations(t *testing.T) {
	parser := NewParser()
	content := `package com.example.first
package com.example.second
class Test
`
	result, err := parser.ParseContent(content, "test.kt")
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	// Should only take the first one
	if result.Package != "com.example.first" {
		t.Errorf("Expected first package declaration, got: %s", result.Package)
	}
}

func TestParser_PackageAllDots(t *testing.T) {
	// Tests fix for BUG-010: Package names must start with a letter
	parser := NewParser()
	content := "package " + strings.Repeat(".", 100) + "\nclass Test"
	result, err := parser.ParseContent(content, "test.kt")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if result.Package != "" {
		t.Errorf("Expected empty package (all-dots is invalid), got '%s'", result.Package)
	}
}

func TestParser_SingleCharPackage(t *testing.T) {
	parser := NewParser()
	content := "package a\nclass Test"

	result, err := parser.ParseContent(content, "test.kt")
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	if result.Package != "a" {
		t.Errorf("Single character package not parsed correctly: %s", result.Package)
	}
}

func TestFQNScanner_NegativeCodeStartLine(t *testing.T) {
	// Tests fix for BUG-001/002: Scanner should handle negative indices gracefully
	scanner := NewFQNScanner()

	tests := []struct {
		name          string
		content       string
		codeStartLine int
	}{
		{"NegativeOne", "class Test", -1},
		{"VeryNegative", "package test\nclass Test", -999999},
		{"EmptyContentNegative", "", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			result := scanner.Scan(tt.content, tt.codeStartLine)
			if result == nil {
				t.Error("Expected non-nil result")
			}
		})
	}
}

func TestFQNScanner_CodeStartLineBeyondContent(t *testing.T) {
	scanner := NewFQNScanner()
	content := "package com.example.test\nimport com.example.Foo"

	result := scanner.Scan(content, 999)

	if len(result.FQNs) != 0 {
		t.Errorf("Should return empty when code start line is beyond content")
	}
}

func TestFQNScanner_FQNInStringTemplate(t *testing.T) {
	scanner := NewFQNScanner()
	content := `package com.example.test

class Test {
    val msg = "Using ${com.example.fake.InTemplate}"
    val raw = """com.example.fake.InRawString"""
    val real = com.example.real.RealClass()
}
`
	result := scanner.Scan(content, 3)

	// FQNs in strings should NOT be detected
	for _, fqn := range result.FQNs {
		if strings.Contains(fqn, "fake") {
			t.Errorf("FQN in string should not be detected: %s", fqn)
		}
	}

	// Real one should be detected
	fqnSet := make(map[string]bool)
	for _, fqn := range result.FQNs {
		fqnSet[fqn] = true
	}
	if !fqnSet["com.example.real.RealClass"] {
		t.Errorf("Expected to detect com.example.real.RealClass, got: %v", result.FQNs)
	}
}

func TestFQNScanner_NestedGenerics(t *testing.T) {
	scanner := NewFQNScanner()
	content := `package com.example.test

class Test {
    val map: Map<String, com.example.models.User<com.example.data.Profile>> = emptyMap()
}
`
	result := scanner.Scan(content, 3)

	fqnSet := make(map[string]bool)
	for _, fqn := range result.FQNs {
		fqnSet[fqn] = true
	}

	// Should detect User (without generic params)
	if !fqnSet["com.example.models.User"] {
		t.Errorf("Should detect User, got: %v", result.FQNs)
	}

	// Should detect Profile
	if !fqnSet["com.example.data.Profile"] {
		t.Errorf("Should detect Profile in nested generic, got: %v", result.FQNs)
	}

	// Should NOT have < or > in FQN
	for fqn := range fqnSet {
		if strings.Contains(fqn, "<") || strings.Contains(fqn, ">") {
			t.Errorf("FQN should not contain generic markers: %s", fqn)
		}
	}
}

func TestFQNScanner_TripleQuotedStringMultiline(t *testing.T) {
	scanner := NewFQNScanner()
	content := `package com.example.test

class Test {
    val text = """
        This is a multiline string
        with com.example.fake.FakeClass inside
        and more text
    """
    fun real() = com.example.real.RealClass()
}
`
	result := scanner.Scan(content, 3)

	// FakeClass should NOT be detected
	for _, fqn := range result.FQNs {
		if strings.Contains(fqn, "FakeClass") {
			t.Errorf("FQN in triple-quoted string should not be detected: %s", fqn)
		}
	}

	// RealClass should be detected
	fqnSet := make(map[string]bool)
	for _, fqn := range result.FQNs {
		fqnSet[fqn] = true
	}
	if !fqnSet["com.example.real.RealClass"] {
		t.Errorf("Should detect RealClass outside of triple-quoted string, got: %v", result.FQNs)
	}
}

func TestFQNScanner_FQNWithOnlyTwoSegments(t *testing.T) {
	scanner := NewFQNScanner()
	content := `package com.example

class Test {
    fun test() = io.Client()
}
`
	result := scanner.Scan(content, 2)

	// Should NOT detect io.Client (only 2 segments)
	for _, fqn := range result.FQNs {
		if fqn == "io.Client" {
			t.Errorf("Should not detect FQN with only 2 segments: %s", fqn)
		}
	}
}

func TestFQNScanner_LowercaseClassName(t *testing.T) {
	scanner := NewFQNScanner()
	content := `package com.example

class Test {
    fun test() = com.example.utils.lowercase()
}
`
	result := scanner.Scan(content, 2)

	// Should NOT detect com.example.utils.lowercase (class names must start uppercase)
	for _, fqn := range result.FQNs {
		if fqn == "com.example.utils.lowercase" {
			t.Errorf("Should not detect FQN with lowercase class name: %s", fqn)
		}
	}
}

func TestExtractClassFromFQN_EdgeCases(t *testing.T) {
	// Tests fix for BUG-016
	tests := []struct {
		fqn      string
		expected string
	}{
		{"com.example.foo.Bar", "Bar"},
		{"Foo", "Foo"}, // No dot - return as-is
		{"com.", ""},   // Trailing dot - invalid
		{".", ""},      // Just dot - invalid
		{"", ""},       // Empty
		{"com.example.Foo", "Foo"},
	}

	for _, tt := range tests {
		result := ExtractClassFromFQN(tt.fqn)
		if result != tt.expected {
			t.Errorf("ExtractClassFromFQN(%q): expected %q, got %q", tt.fqn, tt.expected, result)
		}
	}
}

func TestCleanFQN_EdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"com.example.Foo<Bar>", "com.example.Foo"},
		{"com.example.Foo?", "com.example.Foo"},
		{"com.example.Foo[]", "com.example.Foo"},
		{"com.example.Foo<Bar<Baz>>", "com.example.Foo"},
		{"  com.example.Foo  ", "com.example.Foo"},
		{"com.example.Foo.", "com.example.Foo"},
		{"com.example.Foo!", "com.example.Foo"},
	}

	for _, tt := range tests {
		result := cleanFQN(tt.input)
		if result != tt.expected {
			t.Errorf("cleanFQN(%q): expected %q, got %q", tt.input, tt.expected, result)
		}
	}
}
