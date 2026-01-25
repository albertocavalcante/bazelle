// Package jvm provides common abstractions for JVM language gazelle extensions.
// It unifies patterns shared between Kotlin, Groovy, Java, and Scala extensions
// to reduce code duplication and provide a consistent foundation.
package jvm

import "path/filepath"

// Language represents a JVM programming language.
type Language string

const (
	// Kotlin represents the Kotlin programming language.
	Kotlin Language = "kotlin"

	// Groovy represents the Groovy programming language.
	Groovy Language = "groovy"

	// Java represents the Java programming language.
	Java Language = "java"

	// Scala represents the Scala programming language.
	Scala Language = "scala"
)

// FileExtensions returns the file extensions for this language.
func (l Language) FileExtensions() []string {
	switch l {
	case Kotlin:
		return []string{".kt", ".kts"}
	case Groovy:
		return []string{".groovy"}
	case Java:
		return []string{".java"}
	case Scala:
		return []string{".scala"}
	default:
		return nil
	}
}

// MainSourceDir returns the standard source directory for main sources.
func (l Language) MainSourceDir() string {
	return filepath.Join("src", "main", string(l))
}

// TestSourceDir returns the standard source directory for test sources.
func (l Language) TestSourceDir() string {
	return filepath.Join("src", "test", string(l))
}

// DirectivePrefix returns the prefix used for gazelle directives.
func (l Language) DirectivePrefix() string {
	return string(l)
}

// GlobPatterns returns glob patterns for source files in the given subdirectory.
func (l Language) GlobPatterns(subDir string) []string {
	exts := l.FileExtensions()
	patterns := make([]string, 0, len(exts))
	for _, ext := range exts {
		patterns = append(patterns, filepath.Join(subDir, "**", "*"+ext))
	}
	return patterns
}

// AllLanguages returns all supported JVM languages.
func AllLanguages() []Language {
	return []Language{Kotlin, Groovy, Java, Scala}
}
