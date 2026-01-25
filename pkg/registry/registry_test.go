package registry

import (
	"testing"

	"github.com/albertocavalcante/bazelle/pkg/config"
)

func TestLoadLanguages(t *testing.T) {
	cfg := config.NewConfig()

	// Default config should load proto and go
	languages := LoadLanguages(cfg)

	if len(languages) < 2 {
		t.Errorf("expected at least 2 languages (proto, go), got %d", len(languages))
	}

	// Check that proto and go are loaded
	names := make(map[string]bool)
	for _, lang := range languages {
		names[lang.Name()] = true
	}

	if !names["proto"] {
		t.Error("proto language should be loaded by default")
	}
	if !names["go"] {
		t.Error("go language should be loaded by default")
	}
}

func TestLoadLanguagesWithKotlin(t *testing.T) {
	cfg := config.NewConfig()
	trueVal := true
	cfg.Kotlin.Enabled = &trueVal
	cfg.Languages.Enabled = append(cfg.Languages.Enabled, "kotlin")

	languages := LoadLanguages(cfg)

	names := make(map[string]bool)
	for _, lang := range languages {
		names[lang.Name()] = true
	}

	if !names["kotlin"] {
		t.Error("kotlin language should be loaded when enabled")
	}
}

func TestLoadLanguagesWithPython(t *testing.T) {
	cfg := config.NewConfig()
	trueVal := true
	cfg.Python.Enabled = &trueVal
	cfg.Languages.Enabled = append(cfg.Languages.Enabled, "python")

	languages := LoadLanguages(cfg)

	names := make(map[string]bool)
	for _, lang := range languages {
		names[lang.Name()] = true
	}

	if !names["python"] {
		t.Error("python language should be loaded when enabled")
	}
}

func TestLoadLanguagesByName(t *testing.T) {
	languages := LoadLanguagesByName([]string{"go", "proto"})

	if len(languages) != 2 {
		t.Errorf("expected 2 languages, got %d", len(languages))
	}

	names := make(map[string]bool)
	for _, lang := range languages {
		names[lang.Name()] = true
	}

	if !names["go"] {
		t.Error("go language should be loaded")
	}
	if !names["proto"] {
		t.Error("proto language should be loaded")
	}
}

func TestLoadLanguagesByNameUnknown(t *testing.T) {
	languages := LoadLanguagesByName([]string{"unknown", "go"})

	if len(languages) != 1 {
		t.Errorf("expected 1 language (unknown should be skipped), got %d", len(languages))
	}

	if languages[0].Name() != "go" {
		t.Errorf("expected go, got %s", languages[0].Name())
	}
}

func TestAvailableLanguages(t *testing.T) {
	available := AvailableLanguages()

	if len(available) < 5 {
		t.Errorf("expected at least 5 available languages, got %d", len(available))
	}

	// Check some expected languages
	langSet := make(map[string]bool)
	for _, lang := range available {
		langSet[lang] = true
	}

	expected := []string{"go", "proto", "kotlin", "python", "cc"}
	for _, exp := range expected {
		if !langSet[exp] {
			t.Errorf("expected language %q in available languages", exp)
		}
	}
}

func TestIsLanguageAvailable(t *testing.T) {
	if !IsLanguageAvailable("go") {
		t.Error("go should be available")
	}
	if !IsLanguageAvailable("proto") {
		t.Error("proto should be available")
	}
	if !IsLanguageAvailable("kotlin") {
		t.Error("kotlin should be available")
	}
	if !IsLanguageAvailable("python") {
		t.Error("python should be available")
	}
	if IsLanguageAvailable("unknown") {
		t.Error("unknown should not be available")
	}
}

func TestLanguageOrder(t *testing.T) {
	cfg := config.NewConfig()
	trueVal := true
	cfg.Kotlin.Enabled = &trueVal
	cfg.Python.Enabled = &trueVal
	cfg.CC.Enabled = &trueVal
	cfg.Languages.Enabled = []string{"proto", "go", "kotlin", "python", "cc"}

	languages := LoadLanguages(cfg)

	if len(languages) < 5 {
		t.Errorf("expected at least 5 languages, got %d", len(languages))
		return
	}

	// Proto should always be first
	if languages[0].Name() != "proto" {
		t.Errorf("first language should be proto, got %s", languages[0].Name())
	}
}
