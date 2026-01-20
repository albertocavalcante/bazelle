// Package errcheck provides the errcheck analyzer for nogo.
package errcheck

import "github.com/kisielk/errcheck/errcheck"

// Analyzer detects unchecked errors in Go code.
var Analyzer = errcheck.Analyzer
