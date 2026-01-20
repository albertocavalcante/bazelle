package kotlin

import (
	"testing"
)

func TestKinds(t *testing.T) {
	lang := &kotlinLang{}
	kinds := lang.Kinds()

	// Expected rule kinds
	expectedKinds := []string{
		"kt_jvm_library",
		"kt_jvm_test",
		"kt_jvm_binary",
		"kt_library",
		"kt_test",
	}

	// Verify all expected kinds are present
	for _, kind := range expectedKinds {
		if _, ok := kinds[kind]; !ok {
			t.Errorf("Expected kind '%s' not found", kind)
		}
	}

	// Verify we have the expected number of kinds
	if len(kinds) != len(expectedKinds) {
		t.Errorf("Expected %d kinds, got %d", len(expectedKinds), len(kinds))
	}
}

func TestKinds_KtJvmLibrary(t *testing.T) {
	lang := &kotlinLang{}
	kinds := lang.Kinds()

	info, ok := kinds["kt_jvm_library"]
	if !ok {
		t.Fatal("kt_jvm_library kind not found")
	}

	// Check MatchAny
	if info.MatchAny {
		t.Error("Expected MatchAny to be false")
	}

	// Check NonEmptyAttrs
	if !info.NonEmptyAttrs["srcs"] {
		t.Error("Expected srcs to be in NonEmptyAttrs")
	}

	// Check MergeableAttrs
	if !info.MergeableAttrs["srcs"] {
		t.Error("Expected srcs to be in MergeableAttrs")
	}
	if !info.MergeableAttrs["deps"] {
		t.Error("Expected deps to be in MergeableAttrs")
	}

	// Check ResolveAttrs
	if !info.ResolveAttrs["deps"] {
		t.Error("Expected deps to be in ResolveAttrs")
	}
}

func TestKinds_KtJvmTest(t *testing.T) {
	lang := &kotlinLang{}
	kinds := lang.Kinds()

	info, ok := kinds["kt_jvm_test"]
	if !ok {
		t.Fatal("kt_jvm_test kind not found")
	}

	// Check MatchAny
	if info.MatchAny {
		t.Error("Expected MatchAny to be false")
	}

	// Check NonEmptyAttrs
	if !info.NonEmptyAttrs["srcs"] {
		t.Error("Expected srcs to be in NonEmptyAttrs")
	}

	// Check MergeableAttrs
	if !info.MergeableAttrs["srcs"] {
		t.Error("Expected srcs to be in MergeableAttrs")
	}
	if !info.MergeableAttrs["deps"] {
		t.Error("Expected deps to be in MergeableAttrs")
	}

	// Check ResolveAttrs
	if !info.ResolveAttrs["deps"] {
		t.Error("Expected deps to be in ResolveAttrs")
	}
}

func TestKinds_KtJvmBinary(t *testing.T) {
	lang := &kotlinLang{}
	kinds := lang.Kinds()

	info, ok := kinds["kt_jvm_binary"]
	if !ok {
		t.Fatal("kt_jvm_binary kind not found")
	}

	// Check NonEmptyAttrs
	if !info.NonEmptyAttrs["srcs"] {
		t.Error("Expected srcs to be in NonEmptyAttrs")
	}

	// Check MergeableAttrs
	if !info.MergeableAttrs["srcs"] {
		t.Error("Expected srcs to be in MergeableAttrs")
	}
	if !info.MergeableAttrs["deps"] {
		t.Error("Expected deps to be in MergeableAttrs")
	}

	// Check ResolveAttrs
	if !info.ResolveAttrs["deps"] {
		t.Error("Expected deps to be in ResolveAttrs")
	}
}

func TestKinds_CustomMacros(t *testing.T) {
	lang := &kotlinLang{}
	kinds := lang.Kinds()

	// Check kt_library
	if info, ok := kinds["kt_library"]; !ok {
		t.Error("kt_library kind not found")
	} else {
		if !info.NonEmptyAttrs["srcs"] {
			t.Error("kt_library: Expected srcs to be in NonEmptyAttrs")
		}
		if !info.MergeableAttrs["srcs"] {
			t.Error("kt_library: Expected srcs to be in MergeableAttrs")
		}
		if !info.MergeableAttrs["deps"] {
			t.Error("kt_library: Expected deps to be in MergeableAttrs")
		}
		if !info.ResolveAttrs["deps"] {
			t.Error("kt_library: Expected deps to be in ResolveAttrs")
		}
	}

	// Check kt_test
	if info, ok := kinds["kt_test"]; !ok {
		t.Error("kt_test kind not found")
	} else {
		if !info.NonEmptyAttrs["srcs"] {
			t.Error("kt_test: Expected srcs to be in NonEmptyAttrs")
		}
		if !info.MergeableAttrs["srcs"] {
			t.Error("kt_test: Expected srcs to be in MergeableAttrs")
		}
		if !info.MergeableAttrs["deps"] {
			t.Error("kt_test: Expected deps to be in MergeableAttrs")
		}
		if !info.ResolveAttrs["deps"] {
			t.Error("kt_test: Expected deps to be in ResolveAttrs")
		}
	}
}

func TestKinds_AllHaveConsistentStructure(t *testing.T) {
	lang := &kotlinLang{}
	kinds := lang.Kinds()

	// All kinds should have consistent structure
	for kindName, info := range kinds {
		// All should have srcs in NonEmptyAttrs
		if !info.NonEmptyAttrs["srcs"] {
			t.Errorf("%s: Expected srcs to be in NonEmptyAttrs", kindName)
		}

		// All should have srcs in MergeableAttrs
		if !info.MergeableAttrs["srcs"] {
			t.Errorf("%s: Expected srcs to be in MergeableAttrs", kindName)
		}

		// All should have deps in MergeableAttrs
		if !info.MergeableAttrs["deps"] {
			t.Errorf("%s: Expected deps to be in MergeableAttrs", kindName)
		}

		// All should have deps in ResolveAttrs
		if !info.ResolveAttrs["deps"] {
			t.Errorf("%s: Expected deps to be in ResolveAttrs", kindName)
		}

		// All should have MatchAny = false
		if info.MatchAny {
			t.Errorf("%s: Expected MatchAny to be false", kindName)
		}
	}
}

func TestLoads(t *testing.T) {
	lang := &kotlinLang{}
	loads := lang.Loads()

	if len(loads) != 1 {
		t.Fatalf("Expected 1 load info, got %d", len(loads))
	}

	loadInfo := loads[0]

	// Check the load name
	expectedName := "@rules_kotlin//kotlin:jvm.bzl"
	if loadInfo.Name != expectedName {
		t.Errorf("Expected load name '%s', got '%s'", expectedName, loadInfo.Name)
	}

	// Check the symbols
	expectedSymbols := []string{"kt_jvm_library", "kt_jvm_test", "kt_jvm_binary"}
	if len(loadInfo.Symbols) != len(expectedSymbols) {
		t.Fatalf("Expected %d symbols, got %d", len(expectedSymbols), len(loadInfo.Symbols))
	}

	symbolSet := make(map[string]bool)
	for _, sym := range loadInfo.Symbols {
		symbolSet[sym] = true
	}

	for _, expected := range expectedSymbols {
		if !symbolSet[expected] {
			t.Errorf("Expected symbol '%s' not found in load info", expected)
		}
	}
}

func TestLoads_SymbolsMatchKinds(t *testing.T) {
	lang := &kotlinLang{}
	loads := lang.Loads()
	kinds := lang.Kinds()

	if len(loads) != 1 {
		t.Fatal("Expected exactly 1 load info")
	}

	loadInfo := loads[0]

	// All symbols in Loads should have corresponding entries in Kinds
	for _, symbol := range loadInfo.Symbols {
		if _, ok := kinds[symbol]; !ok {
			t.Errorf("Symbol '%s' in Loads() not found in Kinds()", symbol)
		}
	}
}

func TestKinds_ReturnType(t *testing.T) {
	lang := &kotlinLang{}
	kinds := lang.Kinds()

	// Verify return type is map[string]rule.KindInfo
	if kinds == nil {
		t.Fatal("Expected non-nil map")
	}

	// Verify we can access KindInfo fields
	for _, info := range kinds {
		// These should not panic
		_ = info.MatchAny
		_ = info.NonEmptyAttrs
		_ = info.SubstituteAttrs
		_ = info.MergeableAttrs
		_ = info.ResolveAttrs
	}
}

func TestLoads_ReturnType(t *testing.T) {
	lang := &kotlinLang{}
	loads := lang.Loads()

	// Verify return type is []rule.LoadInfo
	if loads == nil {
		t.Fatal("Expected non-nil slice")
	}

	// Verify we can access LoadInfo fields
	for _, load := range loads {
		// These should not panic
		_ = load.Name
		_ = load.Symbols
	}
}
