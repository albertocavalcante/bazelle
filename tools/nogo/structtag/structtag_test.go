package structtag

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestStructtag(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, Analyzer, "a")
}
