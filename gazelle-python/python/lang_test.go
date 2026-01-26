package python

import (
	"flag"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/language"
)

// ============================================================================
// pythonLang Tests
// ============================================================================

func TestNewLanguage(t *testing.T) {
	lang := NewLanguage()
	if lang == nil {
		t.Fatal("NewLanguage() returned nil")
	}

	// Check it's the correct type
	pl, ok := lang.(*pythonLang)
	if !ok {
		t.Fatal("NewLanguage() did not return *pythonLang")
	}

	if pl.parser == nil {
		t.Error("parser is nil")
	}
}

func TestLanguageName(t *testing.T) {
	lang := NewLanguage()
	name := lang.Name()

	if name != "python" {
		t.Errorf("expected name 'python', got %q", name)
	}
}

func TestLanguageImplementsInterface(t *testing.T) {
	lang := NewLanguage()

	// Verify it implements language.Language interface
	var _ language.Language = lang
}

func TestRegisterFlags(t *testing.T) {
	lang := NewLanguage().(*pythonLang)
	c := &config.Config{
		Exts: make(map[string]interface{}),
	}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	// Should not panic
	lang.RegisterFlags(fs, "update", c)

	// Should have registered the python config
	pc := GetPythonConfig(c)
	if pc == nil {
		t.Error("expected config to be registered")
	}
}

func TestCheckFlags(t *testing.T) {
	lang := NewLanguage().(*pythonLang)
	c := &config.Config{
		Exts: make(map[string]interface{}),
	}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	err := lang.CheckFlags(fs, c)
	if err != nil {
		t.Errorf("CheckFlags returned error: %v", err)
	}
}

func TestPythonNameConstant(t *testing.T) {
	if pythonName != "python" {
		t.Errorf("expected pythonName to be 'python', got %q", pythonName)
	}
}
