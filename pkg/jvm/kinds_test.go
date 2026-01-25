package jvm

import "testing"

func TestStandardLibraryKindInfo(t *testing.T) {
	info := StandardLibraryKindInfo()

	if info.MatchAny {
		t.Error("StandardLibraryKindInfo().MatchAny = true, want false")
	}

	if !info.NonEmptyAttrs["srcs"] {
		t.Error("StandardLibraryKindInfo().NonEmptyAttrs[srcs] = false, want true")
	}

	if !info.MergeableAttrs["srcs"] {
		t.Error("StandardLibraryKindInfo().MergeableAttrs[srcs] = false, want true")
	}

	if !info.MergeableAttrs["deps"] {
		t.Error("StandardLibraryKindInfo().MergeableAttrs[deps] = false, want true")
	}

	if !info.ResolveAttrs["deps"] {
		t.Error("StandardLibraryKindInfo().ResolveAttrs[deps] = false, want true")
	}
}

func TestStandardTestKindInfo(t *testing.T) {
	info := StandardTestKindInfo()

	if info.MatchAny {
		t.Error("StandardTestKindInfo().MatchAny = true, want false")
	}

	if !info.NonEmptyAttrs["srcs"] {
		t.Error("StandardTestKindInfo().NonEmptyAttrs[srcs] = false, want true")
	}
}

func TestSpecTestKindInfo(t *testing.T) {
	info := SpecTestKindInfo()

	if info.MatchAny {
		t.Error("SpecTestKindInfo().MatchAny = true, want false")
	}

	if !info.NonEmptyAttrs["specs"] {
		t.Error("SpecTestKindInfo().NonEmptyAttrs[specs] = false, want true")
	}

	if !info.MergeableAttrs["specs"] {
		t.Error("SpecTestKindInfo().MergeableAttrs[specs] = false, want true")
	}
}

func TestKindInfoSetMerge(t *testing.T) {
	set1 := KindInfoSet{
		"rule_a": StandardLibraryKindInfo(),
		"rule_b": StandardTestKindInfo(),
	}

	set2 := KindInfoSet{
		"rule_c": SpecTestKindInfo(),
		"rule_a": SpecTestKindInfo(), // Override rule_a
	}

	merged := set1.Merge(set2)

	if len(merged) != 3 {
		t.Errorf("Merged set has %d entries, want 3", len(merged))
	}

	// rule_a should be overridden by set2
	if !merged["rule_a"].NonEmptyAttrs["specs"] {
		t.Error("Merged rule_a should have specs from set2")
	}

	// rule_b should be from set1
	if !merged["rule_b"].NonEmptyAttrs["srcs"] {
		t.Error("Merged rule_b should have srcs from set1")
	}

	// rule_c should be from set2
	if !merged["rule_c"].NonEmptyAttrs["specs"] {
		t.Error("Merged rule_c should have specs from set2")
	}
}

func TestStandardKotlinKinds(t *testing.T) {
	kinds := StandardKotlinKinds()

	expectedKinds := []string{
		"kt_jvm_library",
		"kt_jvm_test",
		"kt_jvm_binary",
		"kt_library",
		"kt_test",
	}

	for _, k := range expectedKinds {
		if _, ok := kinds[k]; !ok {
			t.Errorf("StandardKotlinKinds() missing kind %q", k)
		}
	}
}

func TestStandardGroovyKinds(t *testing.T) {
	kinds := StandardGroovyKinds()

	expectedKinds := []string{
		"groovy_library",
		"groovy_test",
		"spock_test",
	}

	for _, k := range expectedKinds {
		if _, ok := kinds[k]; !ok {
			t.Errorf("StandardGroovyKinds() missing kind %q", k)
		}
	}

	// Verify spock_test uses specs attribute
	spockInfo := kinds["spock_test"]
	if !spockInfo.NonEmptyAttrs["specs"] {
		t.Error("spock_test should have specs as NonEmptyAttrs")
	}
}

func TestStandardJavaKinds(t *testing.T) {
	kinds := StandardJavaKinds()

	expectedKinds := []string{
		"java_library",
		"java_test",
		"java_binary",
	}

	for _, k := range expectedKinds {
		if _, ok := kinds[k]; !ok {
			t.Errorf("StandardJavaKinds() missing kind %q", k)
		}
	}
}

func TestStandardScalaKinds(t *testing.T) {
	kinds := StandardScalaKinds()

	expectedKinds := []string{
		"scala_library",
		"scala_test",
		"scala_binary",
	}

	for _, k := range expectedKinds {
		if _, ok := kinds[k]; !ok {
			t.Errorf("StandardScalaKinds() missing kind %q", k)
		}
	}
}
