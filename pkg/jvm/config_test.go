package jvm

import "testing"

func TestNewBaseConfig(t *testing.T) {
	tests := []struct {
		lang             Language
		wantLibraryMacro string
		wantTestMacro    string
		wantEnabled      bool
		wantVisibility   string
	}{
		{Kotlin, "kt_jvm_library", "kt_jvm_test", false, "//visibility:public"},
		{Groovy, "groovy_library", "groovy_test", false, "//visibility:public"},
		{Java, "java_library", "java_test", false, "//visibility:public"},
		{Scala, "scala_library", "scala_test", false, "//visibility:public"},
	}

	for _, tt := range tests {
		t.Run(string(tt.lang), func(t *testing.T) {
			cfg := NewBaseConfig(tt.lang)

			if cfg.LibraryMacro != tt.wantLibraryMacro {
				t.Errorf("LibraryMacro = %q, want %q", cfg.LibraryMacro, tt.wantLibraryMacro)
			}
			if cfg.TestMacro != tt.wantTestMacro {
				t.Errorf("TestMacro = %q, want %q", cfg.TestMacro, tt.wantTestMacro)
			}
			if cfg.Enabled != tt.wantEnabled {
				t.Errorf("Enabled = %v, want %v", cfg.Enabled, tt.wantEnabled)
			}
			if cfg.Visibility != tt.wantVisibility {
				t.Errorf("Visibility = %q, want %q", cfg.Visibility, tt.wantVisibility)
			}
		})
	}
}

func TestBaseConfigClone(t *testing.T) {
	original := BaseConfig{
		Enabled:      true,
		LibraryMacro: "custom_library",
		TestMacro:    "custom_test",
		Visibility:   "//my:visibility",
		LoadPath:     "//:macros.bzl",
	}

	clone := original.CloneBase()

	// Verify the clone has the same values
	if clone.Enabled != original.Enabled {
		t.Errorf("Clone Enabled = %v, want %v", clone.Enabled, original.Enabled)
	}
	if clone.LibraryMacro != original.LibraryMacro {
		t.Errorf("Clone LibraryMacro = %q, want %q", clone.LibraryMacro, original.LibraryMacro)
	}
	if clone.TestMacro != original.TestMacro {
		t.Errorf("Clone TestMacro = %q, want %q", clone.TestMacro, original.TestMacro)
	}
	if clone.Visibility != original.Visibility {
		t.Errorf("Clone Visibility = %q, want %q", clone.Visibility, original.Visibility)
	}
	if clone.LoadPath != original.LoadPath {
		t.Errorf("Clone LoadPath = %q, want %q", clone.LoadPath, original.LoadPath)
	}

	// Verify modifying clone doesn't affect original
	clone.Enabled = false
	clone.LibraryMacro = "other_library"
	if original.Enabled != true {
		t.Error("Modifying clone affected original Enabled")
	}
	if original.LibraryMacro != "custom_library" {
		t.Error("Modifying clone affected original LibraryMacro")
	}
}

func TestBaseConfigGettersSetters(t *testing.T) {
	cfg := &BaseConfig{}

	// Test setters
	cfg.SetEnabled(true)
	cfg.SetLibraryMacro("my_library")
	cfg.SetTestMacro("my_test")
	cfg.SetVisibility("//pkg:visibility")
	cfg.SetLoadPath("//:load.bzl")

	// Test getters
	if !cfg.IsEnabled() {
		t.Error("IsEnabled() = false, want true")
	}
	if cfg.GetLibraryMacro() != "my_library" {
		t.Errorf("GetLibraryMacro() = %q, want %q", cfg.GetLibraryMacro(), "my_library")
	}
	if cfg.GetTestMacro() != "my_test" {
		t.Errorf("GetTestMacro() = %q, want %q", cfg.GetTestMacro(), "my_test")
	}
	if cfg.GetVisibility() != "//pkg:visibility" {
		t.Errorf("GetVisibility() = %q, want %q", cfg.GetVisibility(), "//pkg:visibility")
	}
	if cfg.GetLoadPath() != "//:load.bzl" {
		t.Errorf("GetLoadPath() = %q, want %q", cfg.GetLoadPath(), "//:load.bzl")
	}
}
