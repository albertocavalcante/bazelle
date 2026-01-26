package python

import (
	"testing"
)

func TestIsStdlib(t *testing.T) {
	tests := []struct {
		module   string
		expected bool
	}{
		// Standard library modules
		{"os", true},
		{"sys", true},
		{"json", true},
		{"collections", true},
		{"typing", true},
		{"pathlib", true},
		{"unittest", true},
		{"asyncio", true},
		{"re", true},
		{"datetime", true},
		{"logging", true},
		{"functools", true},
		{"itertools", true},
		{"dataclasses", true},

		// Submodules (should check top-level)
		{"os.path", true},
		{"collections.abc", true},
		{"urllib.parse", true},
		{"xml.etree.ElementTree", true},

		// Non-stdlib modules
		{"numpy", false},
		{"pandas", false},
		{"requests", false},
		{"django", false},
		{"flask", false},
		{"pytest", false},
		{"mypackage", false},
	}

	for _, tt := range tests {
		result := IsStdlib(tt.module)
		if result != tt.expected {
			t.Errorf("IsStdlib(%q) = %v, want %v", tt.module, result, tt.expected)
		}
	}
}

func TestGetStdlibModules(t *testing.T) {
	modules := GetStdlibModules()

	// Should have a reasonable number of modules (Python 3.x has ~200-300)
	if len(modules) < 100 {
		t.Errorf("expected at least 100 stdlib modules, got %d", len(modules))
	}

	// Check some expected modules
	moduleSet := make(map[string]bool)
	for _, m := range modules {
		moduleSet[m] = true
	}

	expectedModules := []string{"os", "sys", "json", "re", "typing", "asyncio"}
	for _, exp := range expectedModules {
		if !moduleSet[exp] {
			t.Errorf("expected module %q not found in stdlib modules", exp)
		}
	}
}

func TestStdlibModuleCount(t *testing.T) {
	count := StdlibModuleCount()

	// Should have a reasonable count
	if count < 100 || count > 500 {
		t.Errorf("stdlib module count %d seems unreasonable (expected 100-500)", count)
	}
}

func TestIsStdlibConcurrent(t *testing.T) {
	// Test that IsStdlib is safe for concurrent access
	done := make(chan bool)

	for range 10 {
		go func() {
			for range 100 {
				IsStdlib("os")
				IsStdlib("numpy")
				IsStdlib("json")
			}
			done <- true
		}()
	}

	for range 10 {
		<-done
	}
}
