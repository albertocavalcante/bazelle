// Package copylock provides the copylock analyzer for nogo.
package copylock

import "golang.org/x/tools/go/analysis/passes/copylock"

// Analyzer detects copied locks that may lead to deadlocks or data races.
var Analyzer = copylock.Analyzer
