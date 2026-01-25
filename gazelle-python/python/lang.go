// Package python provides a Gazelle extension for Python projects.
package python

import (
	"github.com/bazelbuild/bazel-gazelle/language"
)

const pythonName = "python"

// pythonLang implements the language.Language interface for Python.
type pythonLang struct {
	parser *PythonParser
}

// NewLanguage creates a new Python language extension for Gazelle.
func NewLanguage() language.Language {
	return &pythonLang{
		parser: NewParser(),
	}
}

// Name returns the name of the language extension.
func (*pythonLang) Name() string {
	return pythonName
}

var _ language.Language = (*pythonLang)(nil)
