// Package kotlin provides a Gazelle extension for Kotlin projects.
package kotlin

import (
	"github.com/bazelbuild/bazel-gazelle/language"
)

const kotlinName = "kotlin"

// kotlinLang implements the language.Language interface for Kotlin.
type kotlinLang struct {
	parser *KotlinParser
}

// NewLanguage creates a new Kotlin language extension for Gazelle.
func NewLanguage() language.Language {
	return &kotlinLang{
		parser: NewParser(),
	}
}

// Name returns the name of the language extension.
func (*kotlinLang) Name() string {
	return kotlinName
}

var _ language.Language = (*kotlinLang)(nil)
