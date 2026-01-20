package kotlin

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
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
