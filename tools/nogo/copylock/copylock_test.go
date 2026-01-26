package copylock

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestCopylock(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, Analyzer, "a")
}
