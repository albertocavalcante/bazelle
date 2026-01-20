// Package structtag provides the structtag analyzer for nogo.
package structtag

import "golang.org/x/tools/go/analysis/passes/structtag"

// Analyzer validates struct tag formatting and usage.
var Analyzer = structtag.Analyzer
