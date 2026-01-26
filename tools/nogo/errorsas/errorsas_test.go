package errorsas

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestErrorsas(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, Analyzer, "a")
}
