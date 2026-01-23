package tblcalc_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/knaka/tblcalc"
	"github.com/knaka/tblcalc/testdata"
)

func TestExecute_CSV(t *testing.T) {
	// Execute table calculation
	input := strings.NewReader(testdata.Test1CSV)
	var output bytes.Buffer
	err := tblcalc.Execute(input, tblcalc.InputFormatCSV, &output, tblcalc.OutputFormatCSV)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Compare output with expected result
	if output.String() != testdata.Test1ResultCSV {
		t.Errorf("Output mismatch for CSV:\nGot:\n%s\nExpected:\n%s", output.String(), testdata.Test1ResultCSV)
	}
}

func TestExecute_TSV(t *testing.T) {
	// Execute table calculation
	input := strings.NewReader(testdata.Test1TSV)
	var output bytes.Buffer
	err := tblcalc.Execute(input, tblcalc.InputFormatTSV, &output, tblcalc.OutputFormatTSV)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Compare output with expected result
	if output.String() != testdata.Test1ResultTSV {
		t.Errorf("Output mismatch for TSV:\nGot:\n%s\nExpected:\n%s", output.String(), testdata.Test1ResultTSV)
	}
}
