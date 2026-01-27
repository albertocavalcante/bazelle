// Package jvm provides common abstractions for JVM language gazelle extensions.
//
// It unifies patterns shared between Kotlin, Groovy, Java, and Scala extensions
// to reduce code duplication and provide a consistent foundation.
//
// # Extension Mappings
//
// This package defines DETERMINISTIC file extension mappings for JVM languages.
// These mappings are consistent with (but independent of) the mappings in
// langs.Extensions. The duplication exists because this package is used by
// gazelle extensions that don't depend on the bazelle CLI internals.
//
// The mappings are:
//   - Kotlin: .kt, .kts
//   - Groovy: .groovy
//   - Java: .java
//   - Scala: .scala
//
// # Directory Conventions
//
// Standard Maven/Gradle directory layouts are supported:
//   - Main sources: src/main/<language>/
//   - Test sources: src/test/<language>/
package jvm

import "path/filepath"

// Language represents a JVM programming language.
//
// The Language type provides DETERMINISTIC mappings for:
//   - File extensions (FileExtensions)
//   - Source directories (MainSourceDir, TestSourceDir)
//   - Glob patterns (GlobPatterns)
//   - Gazelle directive prefixes (DirectivePrefix)
type Language string

const (
	// Kotlin represents the Kotlin programming language.
	// Extensions: .kt (source), .kts (scripts/build files)
	Kotlin Language = "kotlin"

	// Groovy represents the Groovy programming language.
	// Extensions: .groovy
	Groovy Language = "groovy"

	// Java represents the Java programming language.
	// Extensions: .java
	Java Language = "java"

	// Scala represents the Scala programming language.
	// Extensions: .scala
	Scala Language = "scala"
)

// FileExtensions returns the file extensions for this language.
//
// DETERMINISTIC: The same language always returns the same extensions.
//
// Note: These mappings are consistent with langs.Extensions but defined
// separately to avoid circular dependencies with CLI internals.
//
// Extensions by language:
//   - Kotlin: .kt (source), .kts (scripts)
//   - Groovy: .groovy (standard), .gvy/.gy (short forms), .gsh (shell scripts)
//   - Java: .java
//   - Scala: .scala (source), .sc (Ammonite scripts/worksheets)
func (l Language) FileExtensions() []string {
	switch l {
	case Kotlin:
		return []string{".kt", ".kts"}
	case Groovy:
		return []string{".groovy", ".gvy", ".gy", ".gsh"}
	case Java:
		return []string{".java"}
	case Scala:
		return []string{".scala", ".sc"}
	default:
		return nil
	}
}

// MainSourceDir returns the standard source directory for main sources.
//
// DETERMINISTIC: Returns the conventional Maven/Gradle path.
// Example: Kotlin.MainSourceDir() returns "src/main/kotlin"
func (l Language) MainSourceDir() string {
	return filepath.Join("src", "main", string(l))
}

// TestSourceDir returns the standard source directory for test sources.
//
// DETERMINISTIC: Returns the conventional Maven/Gradle path.
// Example: Kotlin.TestSourceDir() returns "src/test/kotlin"
func (l Language) TestSourceDir() string {
	return filepath.Join("src", "test", string(l))
}

// DirectivePrefix returns the prefix used for gazelle directives.
//
// DETERMINISTIC: Returns the language name as the directive prefix.
// Example: Kotlin.DirectivePrefix() returns "kotlin"
//
// Directives are formatted as: # gazelle:<prefix>_<directive>
func (l Language) DirectivePrefix() string {
	return string(l)
}

// GlobPatterns returns glob patterns for source files in the given subdirectory.
//
// DETERMINISTIC: Given the same subDir, returns the same patterns.
//
// Example: Kotlin.GlobPatterns("src") returns:
//   - "src/**/*.kt"
//   - "src/**/*.kts"
func (l Language) GlobPatterns(subDir string) []string {
	exts := l.FileExtensions()
	patterns := make([]string, 0, len(exts))
	for _, ext := range exts {
		patterns = append(patterns, filepath.Join(subDir, "**", "*"+ext))
	}
	return patterns
}

// AllLanguages returns all supported JVM languages.
//
// DETERMINISTIC: Always returns the same list in the same order.
func AllLanguages() []Language {
	return []Language{Kotlin, Groovy, Java, Scala}
}
