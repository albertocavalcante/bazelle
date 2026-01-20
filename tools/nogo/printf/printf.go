// Package printf provides the printf analyzer for nogo.
package printf

import "golang.org/x/tools/go/analysis/passes/printf"

// Analyzer detects printf-style formatting issues and mismatched arguments.
var Analyzer = printf.Analyzer
