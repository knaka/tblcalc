package tblcalc_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/knaka/tblcalc"
	"github.com/knaka/tblcalc/testdata"
)

func TestExecute_CSV(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "numeric column references",
			input:    testdata.Test1CSV,
			expected: testdata.Test1ResultCSV,
		},
		{
			name:     "header name column references",
			input:    testdata.Test2CSV,
			expected: testdata.Test2ResultCSV,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := strings.NewReader(tt.input)
			var output bytes.Buffer
			err := tblcalc.Execute(input, tblcalc.InputFormatCSV, &output, tblcalc.OutputFormatCSV)
			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}

			if output.String() != tt.expected {
				t.Errorf("Output mismatch:\nGot:\n%s\nExpected:\n%s", output.String(), tt.expected)
			}
		})
	}
}

func TestExecute_TSV(t *testing.T) {
	input := strings.NewReader(testdata.Test1TSV)
	var output bytes.Buffer
	err := tblcalc.Execute(input, tblcalc.InputFormatTSV, &output, tblcalc.OutputFormatTSV)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if output.String() != testdata.Test1ResultTSV {
		t.Errorf("Output mismatch for TSV:\nGot:\n%s\nExpected:\n%s", output.String(), testdata.Test1ResultTSV)
	}
}
