package tblcalc_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/knaka/tblcalc"
)

func TestExecute_CSV(t *testing.T) {
	// Read input CSV file
	inputData, err := os.ReadFile("testdata/test1.csv")
	if err != nil {
		t.Fatalf("failed to read input CSV file: %v", err)
	}

	// Read expected result CSV file
	expectedData, err := os.ReadFile("testdata/test1-result.csv")
	if err != nil {
		t.Fatalf("failed to read expected result CSV file: %v", err)
	}

	// Execute table calculation
	input := bytes.NewReader(inputData)
	var output bytes.Buffer
	err = tblcalc.Execute(input, tblcalc.InputFormatCSV, &output, tblcalc.OutputFormatCSV)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Compare output with expected result
	if output.String() != string(expectedData) {
		t.Errorf("Output mismatch for CSV:\nGot:\n%s\nExpected:\n%s", output.String(), string(expectedData))
	}
}

func TestExecute_TSV(t *testing.T) {
	// Read input TSV file
	inputData, err := os.ReadFile("testdata/test1.tsv")
	if err != nil {
		t.Fatalf("failed to read input TSV file: %v", err)
	}

	// Read expected result TSV file
	expectedData, err := os.ReadFile("testdata/test1-result.tsv")
	if err != nil {
		t.Fatalf("failed to read expected result TSV file: %v", err)
	}

	// Execute table calculation
	input := bytes.NewReader(inputData)
	var output bytes.Buffer
	err = tblcalc.Execute(input, tblcalc.InputFormatTSV, &output, tblcalc.OutputFormatTSV)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Compare output with expected result
	if output.String() != string(expectedData) {
		t.Errorf("Output mismatch for TSV:\nGot:\n%s\nExpected:\n%s", output.String(), string(expectedData))
	}
}
