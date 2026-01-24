package kotlin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

func TestFindKotlinFiles(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()

	// Create src/main/kotlin directory with some files
	mainDir := filepath.Join(tmpDir, "src", "main", "kotlin", "com", "example")
	if err := os.MkdirAll(mainDir, 0755); err != nil {
		t.Fatalf("Failed to create main directory: %v", err)
	}

	// Create some Kotlin files
	files := []string{
		filepath.Join(mainDir, "Main.kt"),
		filepath.Join(mainDir, "Utils.kt"),
	}
	for _, file := range files {
		if err := os.WriteFile(file, []byte("package com.example\n\nclass Test"), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}
	}

	// Test finding files
	result := findKotlinFiles(tmpDir, "src/main/kotlin")

	if len(result) != 2 {
		t.Errorf("Expected 2 files, got %d: %v", len(result), result)
	}

	// Check that paths are relative to base directory
	for _, path := range result {
		if filepath.IsAbs(path) {
			t.Errorf("Expected relative path, got absolute: %s", path)
		}
		if !filepath.HasPrefix(path, "src"+string(filepath.Separator)+"main"+string(filepath.Separator)+"kotlin") {
			t.Errorf("Expected path to start with src/main/kotlin, got: %s", path)
		}
	}
}

func TestFindKotlinFiles_NoDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Try to find files in non-existent directory
	result := findKotlinFiles(tmpDir, "src/main/kotlin")

	if result != nil {
		t.Errorf("Expected nil for non-existent directory, got %v", result)
	}
}

func TestFindKotlinFiles_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create empty directory
	mainDir := filepath.Join(tmpDir, "src", "main", "kotlin")
	if err := os.MkdirAll(mainDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	result := findKotlinFiles(tmpDir, "src/main/kotlin")

	if len(result) != 0 {
		t.Errorf("Expected 0 files in empty directory, got %d: %v", len(result), result)
	}
}

func TestFindKotlinFiles_OnlyKotlinFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory with mixed files
	mainDir := filepath.Join(tmpDir, "src", "main", "kotlin")
	if err := os.MkdirAll(mainDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Create Kotlin and non-Kotlin files
	files := map[string]bool{
		"Test.kt":    true,  // Should be found
		"Main.kt":    true,  // Should be found
		"Script.kts": true,  // Should be found
		"Test.java":  false, // Should be ignored
		"README.md":  false, // Should be ignored
		"build.txt":  false, // Should be ignored
	}

	for filename := range files {
		path := filepath.Join(mainDir, filename)
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", filename, err)
		}
	}

	result := findKotlinFiles(tmpDir, "src/main/kotlin")

	// Should only find .kt files
	expectedCount := 0
	for _, shouldFind := range files {
		if shouldFind {
			expectedCount++
		}
	}

	if len(result) != expectedCount {
		t.Errorf("Expected %d Kotlin files, got %d: %v", expectedCount, len(result), result)
	}

	// Verify all found files are .kt files
	for _, path := range result {
		ext := filepath.Ext(path)
		if ext != ".kt" && ext != ".kts" {
			t.Errorf("Expected only Kotlin files, found: %s", path)
		}
	}
}

func TestGenerateRules_Disabled(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some Kotlin files
	mainDir := filepath.Join(tmpDir, "src", "main", "kotlin")
	if err := os.MkdirAll(mainDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mainDir, "Test.kt"), []byte("class Test"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Create config with Kotlin disabled
	c := &config.Config{
		Exts:     make(map[string]interface{}),
		RepoRoot: tmpDir,
	}
	kc := NewKotlinConfig()
	kc.Enabled = false
	c.Exts[kotlinName] = kc

	lang := NewLanguage()
	args := language.GenerateArgs{
		Config: c,
		Dir:    tmpDir,
	}

	result := lang.GenerateRules(args)

	// Should generate no rules when disabled
	if len(result.Gen) != 0 {
		t.Errorf("Expected no rules when disabled, got %d", len(result.Gen))
	}
}

func TestGenerateRules_NoKotlinFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config with Kotlin enabled
	c := &config.Config{
		Exts:     make(map[string]interface{}),
		RepoRoot: tmpDir,
	}
	kc := NewKotlinConfig()
	kc.Enabled = true
	c.Exts[kotlinName] = kc

	lang := NewLanguage()
	args := language.GenerateArgs{
		Config: c,
		Dir:    tmpDir,
	}

	result := lang.GenerateRules(args)

	// Should generate no rules when no Kotlin files exist
	if len(result.Gen) != 0 {
		t.Errorf("Expected no rules when no Kotlin files, got %d", len(result.Gen))
	}
}

func TestGenerateRules_MainSourcesOnly(t *testing.T) {
	tmpDir := t.TempDir()

	// Create main sources
	mainDir := filepath.Join(tmpDir, "src", "main", "kotlin", "com", "example")
	if err := os.MkdirAll(mainDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	content := `package com.example

class Main {
    fun hello() = "world"
}
`
	if err := os.WriteFile(filepath.Join(mainDir, "Main.kt"), []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Create config
	c := &config.Config{
		Exts:     make(map[string]interface{}),
		RepoRoot: tmpDir,
	}
	kc := NewKotlinConfig()
	kc.Enabled = true
	c.Exts[kotlinName] = kc

	lang := NewLanguage()
	args := language.GenerateArgs{
		Config: c,
		Dir:    tmpDir,
	}

	result := lang.GenerateRules(args)

	// Should generate library rule only
	if len(result.Gen) != 1 {
		t.Fatalf("Expected 1 rule (library), got %d", len(result.Gen))
	}

	libRule := result.Gen[0]
	if libRule.Kind() != "kt_jvm_library" {
		t.Errorf("Expected kt_jvm_library rule, got %s", libRule.Kind())
	}
	if libRule.Name() == "" {
		t.Error("Expected rule to have a name")
	}
}

func TestGenerateRules_TestSourcesOnly(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test sources
	testDir := filepath.Join(tmpDir, "src", "test", "kotlin", "com", "example")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	content := `package com.example

class TestMain {
    fun testHello() = "world"
}
`
	if err := os.WriteFile(filepath.Join(testDir, "TestMain.kt"), []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Create config
	c := &config.Config{
		Exts:     make(map[string]interface{}),
		RepoRoot: tmpDir,
	}
	kc := NewKotlinConfig()
	kc.Enabled = true
	c.Exts[kotlinName] = kc

	lang := NewLanguage()
	args := language.GenerateArgs{
		Config: c,
		Dir:    tmpDir,
	}

	result := lang.GenerateRules(args)

	// Should generate test rule only
	if len(result.Gen) != 1 {
		t.Fatalf("Expected 1 rule (test), got %d", len(result.Gen))
	}

	testRule := result.Gen[0]
	if testRule.Kind() != "kt_jvm_test" {
		t.Errorf("Expected kt_jvm_test rule, got %s", testRule.Kind())
	}
	if testRule.Name() == "" {
		t.Error("Expected rule to have a name")
	}

	// Test-only modules should not reference a missing main library.
	if deps := testRule.AttrStrings("deps"); len(deps) != 0 {
		t.Errorf("Expected no deps for test-only module, got %v", deps)
	}
	if associates := testRule.AttrStrings("associates"); len(associates) != 0 {
		t.Errorf("Expected no associates for test-only module, got %v", associates)
	}
}

func TestGenerateRules_BothMainAndTest(t *testing.T) {
	tmpDir := t.TempDir()

	// Create main sources
	mainDir := filepath.Join(tmpDir, "src", "main", "kotlin")
	if err := os.MkdirAll(mainDir, 0755); err != nil {
		t.Fatalf("Failed to create main directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mainDir, "Main.kt"), []byte("class Main"), 0644); err != nil {
		t.Fatalf("Failed to create main file: %v", err)
	}

	// Create test sources
	testDir := filepath.Join(tmpDir, "src", "test", "kotlin")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "TestMain.kt"), []byte("class TestMain"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create config
	c := &config.Config{
		Exts:     make(map[string]interface{}),
		RepoRoot: tmpDir,
	}
	kc := NewKotlinConfig()
	kc.Enabled = true
	c.Exts[kotlinName] = kc

	lang := NewLanguage()
	args := language.GenerateArgs{
		Config: c,
		Dir:    tmpDir,
	}

	result := lang.GenerateRules(args)

	// Should generate both library and test rules
	if len(result.Gen) != 2 {
		t.Fatalf("Expected 2 rules (library and test), got %d", len(result.Gen))
	}

	// Check that we have both kinds
	kinds := make(map[string]bool)
	for _, r := range result.Gen {
		kinds[r.Kind()] = true
	}

	if !kinds["kt_jvm_library"] {
		t.Error("Expected kt_jvm_library rule")
	}
	if !kinds["kt_jvm_test"] {
		t.Error("Expected kt_jvm_test rule")
	}
}

func TestGenerateRules_CustomMacros(t *testing.T) {
	tmpDir := t.TempDir()

	// Create main sources
	mainDir := filepath.Join(tmpDir, "src", "main", "kotlin")
	if err := os.MkdirAll(mainDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mainDir, "Main.kt"), []byte("class Main"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Create config with custom macros
	c := &config.Config{
		Exts:     make(map[string]interface{}),
		RepoRoot: tmpDir,
	}
	kc := NewKotlinConfig()
	kc.Enabled = true
	kc.LibraryMacro = "custom_kt_library"
	kc.TestMacro = "custom_kt_test"
	c.Exts[kotlinName] = kc

	lang := NewLanguage()
	args := language.GenerateArgs{
		Config: c,
		Dir:    tmpDir,
	}

	result := lang.GenerateRules(args)

	if len(result.Gen) != 1 {
		t.Fatalf("Expected 1 rule, got %d", len(result.Gen))
	}

	rule := result.Gen[0]
	if rule.Kind() != "custom_kt_library" {
		t.Errorf("Expected custom_kt_library rule, got %s", rule.Kind())
	}
}

func TestGenerateRules_RuleAttributes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create main sources
	mainDir := filepath.Join(tmpDir, "src", "main", "kotlin")
	if err := os.MkdirAll(mainDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mainDir, "Main.kt"), []byte("package com.example\n\nclass Main"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Create config
	c := &config.Config{
		Exts:     make(map[string]interface{}),
		RepoRoot: tmpDir,
	}
	kc := NewKotlinConfig()
	kc.Enabled = true
	kc.Visibility = "//custom:visibility"
	c.Exts[kotlinName] = kc

	lang := NewLanguage()
	args := language.GenerateArgs{
		Config: c,
		Dir:    tmpDir,
	}

	result := lang.GenerateRules(args)

	if len(result.Gen) != 1 {
		t.Fatalf("Expected 1 rule, got %d", len(result.Gen))
	}

	genRule := result.Gen[0]

	// Check srcs attribute
	srcsExpr := genRule.Attr("srcs")
	if srcsExpr == nil {
		t.Fatal("Expected srcs attribute to be set")
	}
	if glob, ok := rule.ParseGlobExpr(srcsExpr); ok {
		if len(glob.Patterns) == 0 {
			t.Error("Expected glob patterns to be set for srcs")
		}
	} else {
		t.Error("Expected srcs to be a glob expression")
	}

	// Check visibility attribute
	if vis := genRule.AttrStrings("visibility"); len(vis) == 0 {
		t.Error("Expected visibility attribute to be set")
	} else if vis[0] != "//custom:visibility" {
		t.Errorf("Expected visibility '//custom:visibility', got '%s'", vis[0])
	}
}

func TestGenerateRules_TestRuleAttributes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create main and test sources
	mainDir := filepath.Join(tmpDir, "src", "main", "kotlin")
	if err := os.MkdirAll(mainDir, 0755); err != nil {
		t.Fatalf("Failed to create main directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mainDir, "Main.kt"), []byte("class Main"), 0644); err != nil {
		t.Fatalf("Failed to create main file: %v", err)
	}

	testDir := filepath.Join(tmpDir, "src", "test", "kotlin")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "TestMain.kt"), []byte("package com.example\n\nclass TestMain"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create config
	c := &config.Config{
		Exts:     make(map[string]interface{}),
		RepoRoot: tmpDir,
	}
	kc := NewKotlinConfig()
	kc.Enabled = true
	c.Exts[kotlinName] = kc

	lang := NewLanguage()
	args := language.GenerateArgs{
		Config: c,
		Dir:    tmpDir,
	}

	result := lang.GenerateRules(args)

	// Find the test rule
	var testRule *rule.Rule
	for _, r := range result.Gen {
		if r.Kind() == "kt_jvm_test" {
			testRule = r
			break
		}
	}

	if testRule == nil {
		t.Fatal("Expected to find kt_jvm_test rule")
	}

	// Check that test rule has associates and deps attributes
	if deps := testRule.AttrStrings("deps"); len(deps) == 0 {
		t.Error("Expected test rule to have deps attribute")
	}

	if associates := testRule.AttrStrings("associates"); len(associates) == 0 {
		t.Error("Expected test rule to have associates attribute")
	}
}
