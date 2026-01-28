package tblcalc

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/knaka/go-utils/funcopt"
	"github.com/knaka/tblcalc/testdata"
)

func TestExecute_CSV(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		opts     Options
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
		{
			name:     "exits",
			input:    testdata.Test3CSV,
			expected: testdata.Test3ExitedCSV,
		},
		{
			name:     "does not exit",
			input:    testdata.Test3CSV,
			expected: testdata.Test3NotExitedCSV,
			opts:     []funcopt.Option[tblcalcParams]{WithIgnoreExit(true)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := strings.NewReader(tt.input)
			var output bytes.Buffer
			err := ProcessStream(input, InputFormatCSV, &output, OutputFormatCSV, tt.opts...)
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
	err := ProcessStream(input, InputFormatTSV, &output, OutputFormatTSV)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if output.String() != testdata.Test1ResultTSV {
		t.Errorf("Output mismatch for TSV:\nGot:\n%s\nExpected:\n%s", output.String(), testdata.Test1ResultTSV)
	}
}

func TestExecute_Miller(t *testing.T) {
	tests := []struct {
		name         string
		inputPath    string
		expectedPath string
		opts         Options
	}{
		{
			name:         "multiple scripts",
			inputPath:    filepath.Join("testdata", "mlr-test1.csv"),
			expectedPath: filepath.Join("testdata", "mlr-test1-result.csv"),
		},
		{
			name:         "ignore exit",
			inputPath:    filepath.Join("testdata", "mlr-test1.csv"),
			expectedPath: filepath.Join("testdata", "mlr-test1-result-exit-ignored.csv"),
			opts:         []funcopt.Option[tblcalcParams]{WithIgnoreExit(true)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputData, err := os.ReadFile(tt.inputPath)
			if err != nil {
				t.Fatalf("Failed to read input file: %v", err)
			}

			expectedData, err := os.ReadFile(tt.expectedPath)
			if err != nil {
				t.Fatalf("Failed to read expected file: %v", err)
			}

			input := strings.NewReader(string(inputData))
			var output bytes.Buffer
			err = ProcessStream(input, InputFormatCSV, &output, OutputFormatCSV, tt.opts...)
			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}

			if output.String() != string(expectedData) {
				t.Errorf("Output mismatch:\nGot:\n%s\nExpected:\n%s", output.String(), string(expectedData))
			}

			var output2 bytes.Buffer
			err = ProcessFile(tt.inputPath, InputFormatCSV, &output2, OutputFormatCSV, tt.opts...)
			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}

			if output2.String() != string(expectedData) {
				t.Errorf("Output mismatch:\nGot:\n%s\nExpected:\n%s", output2.String(), string(expectedData))
			}
		})
	}
}
