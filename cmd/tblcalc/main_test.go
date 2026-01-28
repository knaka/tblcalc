package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/knaka/tblcalc"
	"github.com/knaka/tblcalc/testdata"

	//lint:ignore ST1001
	//nolint:staticcheck
	//revive:disable-next-line:dot-imports
	. "github.com/knaka/go-utils"
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

func TestTblcalcEntry_InPlace(t *testing.T) {
	// Create a temporary file with test data
	tempFile, err := os.CreateTemp("", "tblcalc-test-*.csv")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempPath := tempFile.Name()
	defer (func() { Must(os.Remove(tempPath)) })()

	// Write test data to temp file
	Must(tempFile.WriteString(testdata.Test1CSV))
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	Must(tempFile.Close())

	// Setup params with in-place mode
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	params := &tblcalcParams{
		exeName: "tblcalc",
		stdin:   os.Stdin,
		stdout:  &stdout,
		stderr:  &stderr,
		isTerm:  false,
		args:    []string{tempPath},
		verbose: false,
		colored: false,
		inPlace: true,
	}

	// Execute tblcalcEntry
	err = tblcalcEntry(params)
	if err != nil {
		t.Fatalf("tblcalcEntry failed: %v", err)
	}

	// Read the modified file
	result, err := os.ReadFile(tempPath)
	if err != nil {
		t.Fatalf("Failed to read result file: %v", err)
	}

	// Compare with expected result
	if string(result) != testdata.Test1ResultCSV {
		t.Errorf("Output mismatch:\nGot:\n%s\nExpected:\n%s", string(result), testdata.Test1ResultCSV)
	}

	// stdout should be empty in in-place mode
	if stdout.String() != "" {
		t.Errorf("Expected empty stdout in in-place mode, got: %s", stdout.String())
	}
}

func TestTblcalcEntry_InPlace_NoChange(t *testing.T) {
	// Create a temporary file with already-processed data (no formulas to apply)
	tempFile, err := os.CreateTemp("", "tblcalc-test-*.csv")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempPath := tempFile.Name()
	defer (func() { Must(os.Remove(tempPath)) })()

	// Write data that won't change (result data, already processed)
	_, err = tempFile.WriteString(testdata.Test1ResultCSV)
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	Must(tempFile.Close())

	// Get original file info for comparison
	originalInfo, err := os.Stat(tempPath)
	if err != nil {
		t.Fatalf("Failed to stat temp file: %v", err)
	}

	// Setup params with in-place mode
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	params := &tblcalcParams{
		exeName: "tblcalc",
		stdin:   os.Stdin,
		stdout:  &stdout,
		stderr:  &stderr,
		isTerm:  false,
		args:    []string{tempPath},
		verbose: false,
		colored: false,
		inPlace: true,
	}

	// Execute tblcalcEntry
	err = tblcalcEntry(params)
	if err != nil {
		t.Fatalf("tblcalcEntry failed: %v", err)
	}

	// Verify content is unchanged
	result, err := os.ReadFile(tempPath)
	if err != nil {
		t.Fatalf("Failed to read result file: %v", err)
	}
	if string(result) != testdata.Test1ResultCSV {
		t.Errorf("Content should be unchanged:\nGot:\n%s\nExpected:\n%s", string(result), testdata.Test1ResultCSV)
	}

	// Verify file was not rewritten (mod time should be same)
	newInfo, err := os.Stat(tempPath)
	if err != nil {
		t.Fatalf("Failed to stat temp file after processing: %v", err)
	}
	if !originalInfo.ModTime().Equal(newInfo.ModTime()) {
		t.Error("File should not be rewritten when content is unchanged")
	}
}

func TestTblcalcEntry_InPlace_StdinError(t *testing.T) {
	// Setup params with in-place mode and stdin (should fail)
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
		inPlace:              true,
		optForcedInputFormat: &inputFormat,
	}

	// Execute tblcalcEntry - should return error
	err := tblcalcEntry(params)
	if err == nil {
		t.Fatal("Expected error for in-place mode with stdin, but got nil")
	}

	// Verify error message
	if !strings.Contains(err.Error(), "cannot use in-place mode with standard input") {
		t.Errorf("Expected error about in-place mode with stdin, got: %s", err.Error())
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
