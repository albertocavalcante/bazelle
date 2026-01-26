package unusedresult

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestUnusedresult(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, Analyzer, "a")
}
