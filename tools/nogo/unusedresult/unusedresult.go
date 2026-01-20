// Package unusedresult provides the unusedresult analyzer for nogo.
package unusedresult

import "golang.org/x/tools/go/analysis/passes/unusedresult"

// Analyzer detects ignored results from functions that return values.
var Analyzer = unusedresult.Analyzer
