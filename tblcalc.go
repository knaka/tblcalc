// Package tblcalc provides table calculation.
package tblcalc

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"iter"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/knaka/go-utils/funcopt"

	"github.com/knaka/tblcalc/mlr"
	"github.com/knaka/tblcalc/tblfm"

	//lint:ignore ST1001
	//nolint:staticcheck
	//revive:disable-next-line:dot-imports
	. "github.com/knaka/go-utils"
)

// InputFormat represents the format of input data.
type InputFormat int

const (
	// InputFormatCSV indicates CSV (Comma-Separated Values) format.
	InputFormatCSV InputFormat = iota
	// InputFormatTSV indicates TSV (Tab-Separated Values) format.
	InputFormatTSV
)

// OutputFormat represents the format of output data.
type OutputFormat int

const (
	// OutputFormatCSV indicates CSV (Comma-Separated Values) format.
	OutputFormatCSV OutputFormat = iota
	// OutputFormatTSV indicates TSV (Tab-Separated Values) format.
	OutputFormatTSV
)

var commentFormulaRe = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`^#\s*\+TBLFM\s*:\s*(.*)\s*$`)
})

const commentFormulaIdx = 1

var commentScriptRe = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`^#\s*\+(MLR|MILLER)\s*:\s*(.*)\s*$`)
})

const commentScriptIdx = 2

// tblcalcParams holds configuration parameters.
type tblcalcParams struct {
	ignoreExit bool
}

// Options is functional options type
type Options []funcopt.Option[tblcalcParams]

var WithIgnoreExit = funcopt.New(func(params *tblcalcParams, ignoreExit bool) {
	params.ignoreExit = ignoreExit
})

// Execute reads data from reader, applies table formulas found in comment lines,
// and writes the result to writer. Comment lines starting with "# +TBLFM:" contain
// formulas that are applied to the table data. The input and output formats are
// specified by inputFormat and outputFormat parameters.
func Execute(
	reader io.Reader,
	inputFormat InputFormat,
	writer io.Writer,
	outputFormat OutputFormat,
	opts ...funcopt.Option[tblcalcParams],
) (
	err error,
) {
	params := tblcalcParams{
		ignoreExit: false,
	}
	err = funcopt.Apply(&params, opts)
	if err != nil {
		return
	}
	var formulas []string
	var scripts []string
	// Use bufio.Reader to read line by line
	bufReader := bufio.NewReader(reader)
	var commentBlock strings.Builder
	for {
		line, err := bufReader.ReadString('\n')
		if err != nil && err != io.EOF {
			return err
		}
		commentBlock.WriteString(line)
		// Stop processing when we encounter a non-comment line
		if !strings.HasPrefix(line, "#") {
			break
		}
		line = strings.TrimSpace(line)
		if matches := commentFormulaRe().FindStringSubmatch(line); matches != nil {
			formula := matches[commentFormulaIdx]
			formulas = append(formulas, formula)
		} else if matches := commentScriptRe().FindStringSubmatch(line); matches != nil {
			script := matches[commentScriptIdx]
			scripts = append(scripts, script)
		}
	}
	// Reconstruct reader with this line and remaining content
	reader = io.MultiReader(
		strings.NewReader(commentBlock.String()),
		bufReader,
	)
	if len(formulas) > 0 {
		return processWithTBLFMLib(reader, inputFormat, writer, outputFormat, formulas, params.ignoreExit)
	} else if len(scripts) > 0 {
		return processWithMlr(reader, inputFormat, writer, outputFormat, scripts, params.ignoreExit)
	}
	return nil
}

func csvRecordsSeq(
	reader io.Reader,
	onComment func(lineNum int, comment string),
) iter.Seq[[]string] {
	pipeReader, pipeWriter := io.Pipe()
	go (func() {
		defer (func() { Must(pipeWriter.Close()) })()
		scanner := bufio.NewScanner(reader)
		lineNum := 0
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "#") {
				onComment(lineNum, line)
			} else {
				if _, err := fmt.Fprintln(pipeWriter, line); err != nil {
					return
				}
			}
			lineNum++
		}
		if err := scanner.Err(); err != nil {
			pipeWriter.CloseWithError(err)
		}
	})()
	return func(yield func([]string) bool) {
		csvReader := csv.NewReader(pipeReader)
		for {
			record, err := csvReader.Read()
			if err != nil {
				break
			}
			if !yield(record) {
				break
			}
		}
	}
}

func tsvRecordsSeq(
	reader io.Reader,
	onComment func(lineNum int, comment string),
) iter.Seq[[]string] {
	return func(yield func([]string) bool) {
		scanner := bufio.NewScanner(reader)
		lineNum := 0
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "#") {
				onComment(lineNum, line)
			} else {
				record := strings.Split(line, "\t")
				if !yield(record) {
					break
				}
			}
			lineNum++
		}
	}
}

func processWithTBLFMLib(
	reader io.Reader,
	inputFormat InputFormat,
	writer io.Writer,
	outputFormat OutputFormat,
	formulas []string,
	ignoreExit bool,
) (
	err error,
) {
	var table [][]string
	commentLines := make(map[int]string)
	onComment := func(lineNum int, line string) {
		commentLines[lineNum] = line
	}
	var recordsSeq iter.Seq[[]string]
	switch inputFormat {
	case InputFormatCSV:
		recordsSeq = csvRecordsSeq(reader, onComment)
	case InputFormatTSV:
		recordsSeq = tsvRecordsSeq(reader, onComment)
	}
	for record := range recordsSeq {
		table = append(table, record)
	}
	var opts []tblfm.Option
	if ignoreExit {
		opts = append(opts, tblfm.WithIgnoreExit(true))
	}
	// Apply formulas
	if table, err = tblfm.Apply(table, formulas, opts...); err != nil {
		return fmt.Errorf("failed to apply formulas: %v", err)
	}
	// Write output with comments preserved
	switch outputFormat {
	case OutputFormatCSV:
		return writeCSV(writer, table, commentLines)
	case OutputFormatTSV:
		return writeTSV(writer, table, commentLines)
	}
	return
}

func writeCSV(writer io.Writer, table [][]string, commentLines map[int]string) error {
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()
	lineNum := len(table) + len(commentLines)
	tableLineNum := 0
	for i := range lineNum {
		if comment, isComment := commentLines[i]; isComment {
			// Flush CSV writer before writing comment line directly
			csvWriter.Flush()
			if err := csvWriter.Error(); err != nil {
				return err
			}
			// Write comment line
			if _, err := fmt.Fprintln(writer, comment); err != nil {
				return err
			}
		} else {
			// Write table row
			if tableLineNum < len(table) {
				if err := csvWriter.Write(table[tableLineNum]); err != nil {
					return err
				}
				tableLineNum++
			}
		}
	}
	return csvWriter.Error()
}

func writeTSV(writer io.Writer, table [][]string, commentLines map[int]string) error {
	lineNum := len(table) + len(commentLines)
	tableLineNum := 0
	for i := range lineNum {
		if comment, isComment := commentLines[i]; isComment {
			// Write comment line
			if _, err := fmt.Fprintln(writer, comment); err != nil {
				return err
			}
		} else {
			// Write table row as tab-separated values
			if tableLineNum < len(table) {
				line := strings.Join(table[tableLineNum], "\t")
				if _, err := fmt.Fprintln(writer, line); err != nil {
					return err
				}
				tableLineNum++
			}
		}
	}
	return nil
}

func processWithMlr(
	reader io.Reader,
	inputFormat InputFormat,
	writer io.Writer,
	outputFormat OutputFormat,
	scripts []string,
	ignoreExit bool,
) (
	err error,
) {
	// Create a temporary file for Miller processing
	tempFile, err := os.CreateTemp("", "tblcalc-*.csv")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()
	defer (func() {
		Ignore(tempFile.Close())
		Ignore(os.Remove(tempPath))
	})()

	// Write input to temp file
	if _, err = io.Copy(tempFile, reader); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to write to temp file: %w", err)
	}
	if err = tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Determine format strings for Miller
	var inFmt, outFmt string
	switch inputFormat {
	case InputFormatCSV:
		inFmt = "csv"
	case InputFormatTSV:
		inFmt = "tsv"
	}
	switch outputFormat {
	case OutputFormatCSV:
		outFmt = "csv"
	case OutputFormatTSV:
		outFmt = "tsv"
	}

	// Run Miller for each script
	var mlrScripts []string
	for _, script := range scripts {
		if script == "exit" {
			if ignoreExit {
				continue
			} else {
				break
			}
		}
		mlrScripts = append(mlrScripts, script)
	}
	resultFile, err := os.CreateTemp("", "tblcalc-*.csv")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer (func() {
		Ignore(resultFile.Close())
		Ignore(os.Remove(resultFile.Name()))
	})()
	if len(mlrScripts) > 0 {
		err = mlr.Put([]string{tempPath}, mlrScripts, true, inFmt, outFmt, resultFile)
		if err != nil {
			return
		}
	}
	Must(resultFile.Close())

	result := Value(os.Open(resultFile.Name()))

	if _, err = io.Copy(writer, result); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	return nil
}
