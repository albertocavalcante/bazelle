package errcheck

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestErrcheck(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, Analyzer, "a")
}
