package loggercheck_test

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/jlewi/roboweb/tracecheck"
)

// N.B. Before running the tests you need to
// cd testdata/src/a && run
// go mod vendor
// This downloads the dependencies for the test data.

type dummyTestingErrorf struct {
	*testing.T
}

func (t dummyTestingErrorf) Errorf(format string, args ...interface{}) {}

func TestLinter(t *testing.T) {
	// TestData returns the effective filename of the program's "testdata" directory
	testdata := analysistest.TestData()

	testCases := []struct {
		name      string
		patterns  string
		flags     []string
		wantError string
	}{
		{
			name:     "all",
			patterns: "a/all",
			flags:    []string{""},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			a := loggercheck.NewAnalyzer()
			err := a.Flags.Parse(tc.flags)
			require.NoError(t, err)

			var result []*analysistest.Result
			if tc.wantError != "" {
				result = analysistest.Run(&dummyTestingErrorf{t}, testdata, a, tc.patterns)
			} else {
				result = analysistest.Run(t, testdata, a, tc.patterns)
			}
			require.Len(t, result, 1)

			if tc.wantError != "" {
				assert.Error(t, result[0].Err)
				assert.ErrorContains(t, result[0].Err, tc.wantError)
			}
		})
	}
}

func TestLinterFix(t *testing.T) {
	testdata := analysistest.TestData()

	testCases := []struct {
		name      string
		dir  string
	}{
		{
			name:     "fix_import",
			dir: "a/fix_import",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			a := loggercheck.NewAnalyzer()
			analysistest.RunWithSuggestedFixes(t, testdata, a, tc.dir)
		})
	}
}

