package lostcancel

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestLostcancel(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, Analyzer, "a")
}
