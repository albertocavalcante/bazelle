// Package errorsas provides the errorsas analyzer for nogo.
package errorsas

import "golang.org/x/tools/go/analysis/passes/errorsas"

// Analyzer ensures errors are asserted with errors.As instead of type assertions.
var Analyzer = errorsas.Analyzer
