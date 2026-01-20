// Package unmarshal provides the unmarshal analyzer for nogo.
package unmarshal

import "golang.org/x/tools/go/analysis/passes/unmarshal"

// Analyzer detects invalid or ineffective unmarshalling into non-pointer values.
var Analyzer = unmarshal.Analyzer
