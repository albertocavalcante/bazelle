package jvm

import "github.com/bazelbuild/bazel-gazelle/rule"

// StandardLibraryKindInfo returns the standard KindInfo for library rules.
// This is suitable for java_library, kt_jvm_library, groovy_library, etc.
func StandardLibraryKindInfo() rule.KindInfo {
	return rule.KindInfo{
		MatchAny:        false,
		NonEmptyAttrs:   map[string]bool{"srcs": true},
		SubstituteAttrs: map[string]bool{},
		MergeableAttrs:  map[string]bool{"srcs": true, "deps": true},
		ResolveAttrs:    map[string]bool{"deps": true},
	}
}

// StandardTestKindInfo returns the standard KindInfo for test rules.
// This is suitable for java_test, kt_jvm_test, groovy_test, etc.
func StandardTestKindInfo() rule.KindInfo {
	return rule.KindInfo{
		MatchAny:        false,
		NonEmptyAttrs:   map[string]bool{"srcs": true},
		SubstituteAttrs: map[string]bool{},
		MergeableAttrs:  map[string]bool{"srcs": true, "deps": true},
		ResolveAttrs:    map[string]bool{"deps": true},
	}
}

// StandardBinaryKindInfo returns the standard KindInfo for binary rules.
// This is suitable for java_binary, kt_jvm_binary, etc.
func StandardBinaryKindInfo() rule.KindInfo {
	return rule.KindInfo{
		MatchAny:        false,
		NonEmptyAttrs:   map[string]bool{"srcs": true},
		SubstituteAttrs: map[string]bool{},
		MergeableAttrs:  map[string]bool{"srcs": true, "deps": true},
		ResolveAttrs:    map[string]bool{"deps": true},
	}
}

// SpecTestKindInfo returns the KindInfo for spec-based test rules (like Spock).
// Uses "specs" as the primary source attribute instead of "srcs".
func SpecTestKindInfo() rule.KindInfo {
	return rule.KindInfo{
		MatchAny:        false,
		NonEmptyAttrs:   map[string]bool{"specs": true},
		SubstituteAttrs: map[string]bool{},
		MergeableAttrs:  map[string]bool{"specs": true, "deps": true},
		ResolveAttrs:    map[string]bool{"deps": true},
	}
}

// KindInfoSet holds KindInfo for multiple rule kinds.
type KindInfoSet map[string]rule.KindInfo

// Merge combines multiple KindInfoSets into one.
func (s KindInfoSet) Merge(other KindInfoSet) KindInfoSet {
	result := make(KindInfoSet)
	for k, v := range s {
		result[k] = v
	}
	for k, v := range other {
		result[k] = v
	}
	return result
}

// StandardKotlinKinds returns the standard KindInfo map for Kotlin.
func StandardKotlinKinds() KindInfoSet {
	return KindInfoSet{
		"kt_jvm_library": StandardLibraryKindInfo(),
		"kt_jvm_test":    StandardTestKindInfo(),
		"kt_jvm_binary":  StandardBinaryKindInfo(),
		"kt_library":     StandardLibraryKindInfo(),
		"kt_test":        StandardTestKindInfo(),
	}
}

// StandardGroovyKinds returns the standard KindInfo map for Groovy.
func StandardGroovyKinds() KindInfoSet {
	return KindInfoSet{
		"groovy_library": StandardLibraryKindInfo(),
		"groovy_test":    StandardTestKindInfo(),
		"spock_test":     SpecTestKindInfo(),
	}
}

// StandardJavaKinds returns the standard KindInfo map for Java.
func StandardJavaKinds() KindInfoSet {
	return KindInfoSet{
		"java_library": StandardLibraryKindInfo(),
		"java_test":    StandardTestKindInfo(),
		"java_binary":  StandardBinaryKindInfo(),
	}
}

// StandardScalaKinds returns the standard KindInfo map for Scala.
func StandardScalaKinds() KindInfoSet {
	return KindInfoSet{
		"scala_library": StandardLibraryKindInfo(),
		"scala_test":    StandardTestKindInfo(),
		"scala_binary":  StandardBinaryKindInfo(),
	}
}
