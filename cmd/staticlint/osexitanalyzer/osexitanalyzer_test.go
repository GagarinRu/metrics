package osexitanalyzer_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/GagarinRu/metrics/cmd/staticlint/osexitanalyzer"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, osexitanalyzer.Analyzer, "a")
}
