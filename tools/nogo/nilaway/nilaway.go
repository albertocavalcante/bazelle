// Package nilaway provides the nilaway analyzer for nogo.
package nilaway

import "go.uber.org/nilaway"

// Analyzer detects potential nil pointer dereferences.
var Analyzer = nilaway.Analyzer
