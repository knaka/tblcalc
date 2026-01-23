package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/knaka/tblcalc"
	"github.com/knaka/tblcalc/testdata"
)

var projectTopDirPath = filepath.Join("..", "..")

func TestTblcalcEntry_WithCSVFile(t *testing.T) {
	// Setup params with testdata/test1.csv as argument
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	params := &tblcalcParams{
		exeName: "tblcalc",
		stdin:   os.Stdin,
		stdout:  &stdout,
		stderr:  &stderr,
		isTerm:  false,
		args:    []string{filepath.Join(projectTopDirPath, "testdata", "test1.csv")},
		verbose: false,
		colored: false,
		inPlace: false,
	}

	// Execute tblcalcEntry
	err := tblcalcEntry(params)
	if err != nil {
		t.Fatalf("tblcalcEntry failed: %v", err)
	}

	// Compare output with expected result
	if stdout.String() != testdata.Test1ResultCSV {
		t.Errorf("Output mismatch:\nGot:\n%s\nExpected:\n%s", stdout.String(), testdata.Test1ResultCSV)
	}
}

func TestTblcalcEntry_WithStdin(t *testing.T) {
	// Setup params with stdin input
	stdin := strings.NewReader(testdata.Test1CSV)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	inputFormat := tblcalc.InputFormatCSV
	params := &tblcalcParams{
		exeName:              "tblcalc",
		stdin:                stdin,
		stdout:               &stdout,
		stderr:               &stderr,
		isTerm:               false,
		args:                 []string{stdinFileName},
		verbose:              false,
		colored:              false,
		inPlace:              false,
		optForcedInputFormat: &inputFormat,
	}

	// Execute tblcalcEntry
	err := tblcalcEntry(params)
	if err != nil {
		t.Fatalf("tblcalcEntry failed: %v", err)
	}

	// Compare output with expected result
	if stdout.String() != testdata.Test1ResultCSV {
		t.Errorf("Output mismatch:\nGot:\n%s\nExpected:\n%s", stdout.String(), testdata.Test1ResultCSV)
	}
}

func TestTblcalcEntry_NonExistentFile(t *testing.T) {
	// Setup params with non-existent file
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	nonExistentFile := filepath.Join(projectTopDirPath, "testdata", "non-existent-file.csv")
	params := &tblcalcParams{
		exeName: "tblcalc",
		stdin:   os.Stdin,
		stdout:  &stdout,
		stderr:  &stderr,
		isTerm:  false,
		args:    []string{nonExistentFile},
		verbose: false,
		colored: false,
		inPlace: false,
	}

	// Execute tblcalcEntry - should return error
	err := tblcalcEntry(params)
	if err == nil {
		t.Fatal("Expected error for non-existent file, but got nil")
	}

	// Verify error message contains information about failed file opening
	errMsg := err.Error()
	if !strings.Contains(errMsg, "failed to open input file") {
		t.Errorf("Expected error message to contain 'failed to open input file', got: %s", errMsg)
	}
}
