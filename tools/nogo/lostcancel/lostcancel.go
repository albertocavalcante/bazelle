// Package lostcancel provides the lostcancel analyzer for nogo.
package lostcancel

import "golang.org/x/tools/go/analysis/passes/lostcancel"

// Analyzer detects context.Context cancel functions that are not called.
var Analyzer = lostcancel.Analyzer
