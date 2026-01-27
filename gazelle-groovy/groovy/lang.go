// Package groovy provides a Gazelle extension for Groovy projects.
package groovy

import "github.com/bazelbuild/bazel-gazelle/language"

const groovyName = "groovy"

// groovyLang implements the language.Language interface for Groovy.
type groovyLang struct {
	// parser provides HEURISTIC parsing of Groovy source files.
	parser *GroovyParser
}

// NewLanguage creates a new Groovy language extension for Gazelle.
func NewLanguage() language.Language {
	return &groovyLang{
		parser: NewParser(),
	}
}

// Name returns the name of the language extension.
func (*groovyLang) Name() string {
	return groovyName
}

var _ language.Language = (*groovyLang)(nil)
