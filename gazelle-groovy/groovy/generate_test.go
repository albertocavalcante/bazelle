package groovy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

func TestFindGroovyFiles(t *testing.T) {
	tmpDir := t.TempDir()

	mainDir := filepath.Join(tmpDir, "src", "main", "groovy", "com", "example")
	if err := os.MkdirAll(mainDir, 0o755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	files := []string{
		filepath.Join(mainDir, "Main.groovy"),
		filepath.Join(mainDir, "Helper.groovy"),
		filepath.Join(mainDir, "Helper.java"),
		filepath.Join(mainDir, "README.md"),
	}
	for _, file := range files {
		if err := os.WriteFile(file, []byte("content"), 0o644); err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}
	}

	result := findGroovyFiles(tmpDir, "src/main/groovy")
	if len(result) != 2 {
		t.Fatalf("Expected 2 Groovy files, got %d: %v", len(result), result)
	}

	for _, path := range result {
		if filepath.IsAbs(path) {
			t.Errorf("Expected relative path, got absolute: %s", path)
		}
		if filepath.Ext(path) != ".groovy" {
			t.Errorf("Expected only .groovy files, got: %s", path)
		}
	}
}

func TestFindGroovyFiles_NoDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	result := findGroovyFiles(tmpDir, "src/main/groovy")
	if result != nil {
		t.Errorf("Expected nil for non-existent directory, got %v", result)
	}
}

func TestGenerateRules_Disabled(t *testing.T) {
	tmpDir := t.TempDir()

	mainDir := filepath.Join(tmpDir, "src", "main", "groovy")
	if err := os.MkdirAll(mainDir, 0o755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mainDir, "Test.groovy"), []byte("class Test"), 0o644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	c := &config.Config{
		Exts:     make(map[string]interface{}),
		RepoRoot: tmpDir,
	}
	gc := NewGroovyConfig()
	gc.Enabled = false
	c.Exts[groovyName] = gc

	lang := NewLanguage()
	args := language.GenerateArgs{
		Config: c,
		Dir:    tmpDir,
	}

	result := lang.GenerateRules(args)
	if len(result.Gen) != 0 {
		t.Fatalf("Expected no rules when disabled, got %d", len(result.Gen))
	}
}

func TestGenerateRules_MainSourcesOnly(t *testing.T) {
	tmpDir := t.TempDir()

	mainDir := filepath.Join(tmpDir, "src", "main", "groovy")
	if err := os.MkdirAll(mainDir, 0o755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mainDir, "Main.groovy"), []byte("class Main"), 0o644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	c := &config.Config{
		Exts:     make(map[string]interface{}),
		RepoRoot: tmpDir,
	}
	gc := NewGroovyConfig()
	gc.Enabled = true
	c.Exts[groovyName] = gc

	lang := NewLanguage()
	args := language.GenerateArgs{
		Config: c,
		Dir:    tmpDir,
	}

	result := lang.GenerateRules(args)
	if len(result.Gen) != 1 {
		t.Fatalf("Expected 1 rule, got %d", len(result.Gen))
	}

	libRule := result.Gen[0]
	if libRule.Kind() != "groovy_library" {
		t.Errorf("Expected groovy_library, got %s", libRule.Kind())
	}

	srcsExpr := libRule.Attr("srcs")
	if srcsExpr == nil {
		t.Fatal("Expected srcs to be set")
	}
	if glob, ok := rule.ParseGlobExpr(srcsExpr); ok {
		if len(glob.Patterns) == 0 {
			t.Error("Expected glob patterns for srcs")
		}
	} else {
		t.Error("Expected srcs to be a glob expression")
	}
}

func TestGenerateRules_TestSourcesOnly_GroovyTest(t *testing.T) {
	tmpDir := t.TempDir()

	testDir := filepath.Join(tmpDir, "src", "test", "groovy")
	if err := os.MkdirAll(testDir, 0o755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "FooTest.groovy"), []byte("class FooTest {}"), 0o644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	c := &config.Config{
		Exts:     make(map[string]interface{}),
		RepoRoot: tmpDir,
	}
	gc := NewGroovyConfig()
	gc.Enabled = true
	c.Exts[groovyName] = gc

	lang := NewLanguage()
	args := language.GenerateArgs{
		Config: c,
		Dir:    tmpDir,
	}

	result := lang.GenerateRules(args)
	if len(result.Gen) != 1 {
		t.Fatalf("Expected 1 rule, got %d", len(result.Gen))
	}

	testRule := result.Gen[0]
	if testRule.Kind() != "groovy_test" {
		t.Errorf("Expected groovy_test, got %s", testRule.Kind())
	}

	srcsExpr := testRule.Attr("srcs")
	if srcsExpr == nil {
		t.Fatal("Expected srcs to be set")
	}
	if glob, ok := rule.ParseGlobExpr(srcsExpr); ok {
		if len(glob.Patterns) == 0 {
			t.Error("Expected glob patterns for srcs")
		}
	} else {
		t.Error("Expected srcs to be a glob expression")
	}

	if deps := testRule.AttrStrings("deps"); len(deps) != 0 {
		t.Errorf("Expected no deps without main sources, got %v", deps)
	}
}

func TestGenerateRules_TestSourcesOnly_SpockByFilename(t *testing.T) {
	tmpDir := t.TempDir()

	testDir := filepath.Join(tmpDir, "src", "test", "groovy")
	if err := os.MkdirAll(testDir, 0o755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "FooSpec.groovy"), []byte("class FooSpec extends Specification {}"), 0o644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	c := &config.Config{
		Exts:     make(map[string]interface{}),
		RepoRoot: tmpDir,
	}
	gc := NewGroovyConfig()
	gc.Enabled = true
	c.Exts[groovyName] = gc

	lang := NewLanguage()
	args := language.GenerateArgs{
		Config: c,
		Dir:    tmpDir,
	}

	result := lang.GenerateRules(args)
	if len(result.Gen) != 1 {
		t.Fatalf("Expected 1 rule, got %d", len(result.Gen))
	}

	testRule := result.Gen[0]
	if testRule.Kind() != "spock_test" {
		t.Errorf("Expected spock_test, got %s", testRule.Kind())
	}

	specsExpr := testRule.Attr("specs")
	if specsExpr == nil {
		t.Fatal("Expected specs to be set")
	}
	if glob, ok := rule.ParseGlobExpr(specsExpr); ok {
		if len(glob.Patterns) != 2 {
			t.Fatalf("Expected 2 spec patterns, got %v", glob.Patterns)
		}
	} else {
		t.Error("Expected specs to be a glob expression")
	}

	groovySrcsExpr := testRule.Attr("groovy_srcs")
	if groovySrcsExpr == nil {
		t.Fatal("Expected groovy_srcs to be set")
	}
	if glob, ok := rule.ParseGlobExpr(groovySrcsExpr); ok {
		if len(glob.Excludes) == 0 {
			t.Fatal("Expected excludes on groovy_srcs to avoid specs")
		}
	} else {
		t.Error("Expected groovy_srcs to be a glob expression")
	}
}

func TestGenerateRules_TestSourcesOnly_SpockForced(t *testing.T) {
	tmpDir := t.TempDir()

	testDir := filepath.Join(tmpDir, "src", "test", "groovy")
	if err := os.MkdirAll(testDir, 0o755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "FooTest.groovy"), []byte("class FooTest {}"), 0o644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	c := &config.Config{
		Exts:     make(map[string]interface{}),
		RepoRoot: tmpDir,
	}
	gc := NewGroovyConfig()
	gc.Enabled = true
	gc.TestMacro = gc.SpockTestMacro
	c.Exts[groovyName] = gc

	lang := NewLanguage()
	args := language.GenerateArgs{
		Config: c,
		Dir:    tmpDir,
	}

	result := lang.GenerateRules(args)
	if len(result.Gen) != 1 {
		t.Fatalf("Expected 1 rule, got %d", len(result.Gen))
	}

	testRule := result.Gen[0]
	if testRule.Kind() != "spock_test" {
		t.Errorf("Expected spock_test, got %s", testRule.Kind())
	}

	specsExpr := testRule.Attr("specs")
	if specsExpr == nil {
		t.Fatal("Expected specs to be set")
	}
	if glob, ok := rule.ParseGlobExpr(specsExpr); ok {
		if len(glob.Patterns) != 1 || glob.Patterns[0] != testGroovyPattern {
			t.Fatalf("Expected specs to include %q, got %v", testGroovyPattern, glob.Patterns)
		}
	} else {
		t.Error("Expected specs to be a glob expression")
	}
}

func TestGenerateRules_BothMainAndTest(t *testing.T) {
	tmpDir := t.TempDir()

	mainDir := filepath.Join(tmpDir, "src", "main", "groovy")
	if err := os.MkdirAll(mainDir, 0o755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mainDir, "Main.groovy"), []byte("class Main"), 0o644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	testDir := filepath.Join(tmpDir, "src", "test", "groovy")
	if err := os.MkdirAll(testDir, 0o755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "MainTest.groovy"), []byte("class MainTest {}"), 0o644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	c := &config.Config{
		Exts:     make(map[string]interface{}),
		RepoRoot: tmpDir,
	}
	gc := NewGroovyConfig()
	gc.Enabled = true
	c.Exts[groovyName] = gc

	lang := NewLanguage()
	args := language.GenerateArgs{
		Config: c,
		Dir:    tmpDir,
	}

	result := lang.GenerateRules(args)
	if len(result.Gen) != 2 {
		t.Fatalf("Expected 2 rules, got %d", len(result.Gen))
	}

	var testRule *rule.Rule
	for _, r := range result.Gen {
		if r.Kind() == "groovy_test" {
			testRule = r
			break
		}
	}
	if testRule == nil {
		t.Fatal("Expected to find groovy_test rule")
	}
	if deps := testRule.AttrStrings("deps"); len(deps) == 0 {
		t.Error("Expected test rule deps to include main library")
	}
}

func TestGenerateRules_CustomMacros(t *testing.T) {
	tmpDir := t.TempDir()

	mainDir := filepath.Join(tmpDir, "src", "main", "groovy")
	if err := os.MkdirAll(mainDir, 0o755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mainDir, "Main.groovy"), []byte("class Main"), 0o644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	testDir := filepath.Join(tmpDir, "src", "test", "groovy")
	if err := os.MkdirAll(testDir, 0o755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "MainSpec.groovy"), []byte("class MainSpec extends Specification {}"), 0o644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	c := &config.Config{
		Exts:     make(map[string]interface{}),
		RepoRoot: tmpDir,
	}
	gc := NewGroovyConfig()
	gc.Enabled = true
	gc.LibraryMacro = "custom_groovy_library"
	gc.TestMacro = "custom_groovy_test"
	gc.SpockTestMacro = "custom_spock_test"
	c.Exts[groovyName] = gc

	lang := NewLanguage()
	args := language.GenerateArgs{
		Config: c,
		Dir:    tmpDir,
	}

	result := lang.GenerateRules(args)
	if len(result.Gen) != 2 {
		t.Fatalf("Expected 2 rules, got %d", len(result.Gen))
	}

	kinds := make(map[string]bool)
	for _, r := range result.Gen {
		kinds[r.Kind()] = true
	}
	if !kinds["custom_groovy_library"] {
		t.Error("Expected custom groovy library rule")
	}
	if !kinds["custom_spock_test"] {
		t.Error("Expected custom spock test rule")
	}
}
