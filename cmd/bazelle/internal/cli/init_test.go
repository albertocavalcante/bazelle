package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectDependencies_Go(t *testing.T) {
	deps := collectDependencies([]string{"go"})

	hasGazelle := false
	hasRulesGo := false
	for _, dep := range deps {
		if dep.name == "gazelle" {
			hasGazelle = true
		}
		if dep.name == "rules_go" {
			hasRulesGo = true
		}
	}

	if !hasGazelle {
		t.Error("collectDependencies(go) missing gazelle")
	}
	if !hasRulesGo {
		t.Error("collectDependencies(go) missing rules_go")
	}
}

func TestCollectDependencies_Kotlin(t *testing.T) {
	deps := collectDependencies([]string{"kotlin"})

	required := []string{"gazelle", "rules_go", "rules_kotlin"}
	for _, req := range required {
		found := false
		for _, dep := range deps {
			if dep.name == req {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("collectDependencies(kotlin) missing %s", req)
		}
	}
}

func TestCollectDependencies_NoDuplicates(t *testing.T) {
	deps := collectDependencies([]string{"go", "kotlin", "python"})

	seen := make(map[string]bool)
	for _, dep := range deps {
		if seen[dep.name] {
			t.Errorf("collectDependencies has duplicate: %s", dep.name)
		}
		seen[dep.name] = true
	}
}

func TestGenerateModuleContent(t *testing.T) {
	deps := []dependency{
		{name: "gazelle", version: "0.47.0"},
		{name: "rules_go", version: "0.59.0"},
	}

	content := generateModuleContent("myproject", deps)

	if !strings.Contains(content, `name = "myproject"`) {
		t.Error("module content missing module name")
	}
	if !strings.Contains(content, `bazel_dep(name = "gazelle"`) {
		t.Error("module content missing gazelle dep")
	}
	if !strings.Contains(content, `bazel_dep(name = "rules_go"`) {
		t.Error("module content missing rules_go dep")
	}
}

func TestGenerateBuildContent(t *testing.T) {
	content := generateBuildContent()

	if !strings.Contains(content, "@gazelle//:def.bzl") {
		t.Error("build content missing gazelle load")
	}
	if !strings.Contains(content, "gazelle:prefix") {
		t.Error("build content missing gazelle prefix directive")
	}
	if !strings.Contains(content, `name = "gazelle"`) {
		t.Error("build content missing gazelle target")
	}
}

func TestRunInitDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	moduleFile := filepath.Join(tmpDir, "MODULE.bazel")
	buildFile := filepath.Join(tmpDir, "BUILD.bazel")

	// Should not error on dry run
	err := runInitDryRun(false, false, moduleFile, buildFile, "module content", "build content")
	if err != nil {
		t.Errorf("runInitDryRun() error = %v", err)
	}

	// Files should not be created
	if fileExists(moduleFile) {
		t.Error("dry run created MODULE.bazel")
	}
	if fileExists(buildFile) {
		t.Error("dry run created BUILD.bazel")
	}
}

func TestRunInitApply(t *testing.T) {
	tmpDir := t.TempDir()
	moduleFile := filepath.Join(tmpDir, "MODULE.bazel")
	buildFile := filepath.Join(tmpDir, "BUILD.bazel")

	err := runInitApply(false, false, moduleFile, buildFile, "module content\n", "build content\n")
	if err != nil {
		t.Errorf("runInitApply() error = %v", err)
	}

	// Files should be created
	if !fileExists(moduleFile) {
		t.Error("apply did not create MODULE.bazel")
	}
	if !fileExists(buildFile) {
		t.Error("apply did not create BUILD.bazel")
	}

	// Check content
	moduleContent, _ := os.ReadFile(moduleFile)
	if string(moduleContent) != "module content\n" {
		t.Errorf("MODULE.bazel content = %q, want %q", moduleContent, "module content\n")
	}
}

func TestRunInitApply_ExistingFiles(t *testing.T) {
	tmpDir := t.TempDir()
	moduleFile := filepath.Join(tmpDir, "MODULE.bazel")
	buildFile := filepath.Join(tmpDir, "BUILD.bazel")

	// Create existing files
	os.WriteFile(moduleFile, []byte("existing module"), 0o644)
	os.WriteFile(buildFile, []byte("existing build"), 0o644)

	err := runInitApply(true, true, moduleFile, buildFile, "new content", "new content")
	if err != nil {
		t.Errorf("runInitApply() error = %v", err)
	}

	// Files should not be overwritten
	moduleContent, _ := os.ReadFile(moduleFile)
	if string(moduleContent) != "existing module" {
		t.Error("apply overwrote existing MODULE.bazel")
	}
}
