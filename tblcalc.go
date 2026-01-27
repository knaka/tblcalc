// Package tblcalc provides table calculation.
package tblcalc

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"iter"
	"regexp"
	"strings"
	"sync"

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

var reCommentFormula = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`^#\s*\+TBLFM\s*:\s*(.*)\s*$`)
})

const idxFormula = 1

// Execute reads data from reader, applies table formulas found in comment lines,
// and writes the result to writer. Comment lines starting with "# +TBLFM:" contain
// formulas that are applied to the table data. The input and output formats are
// specified by inputFormat and outputFormat parameters.
func Execute(
	reader io.Reader,
	inputFormat InputFormat,
	writer io.Writer,
	outputFormat OutputFormat,
) (
	err error,
) {
	var formulas []string
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
		if matches := reCommentFormula().FindStringSubmatch(line); matches != nil {
			formula := matches[idxFormula]
			formulas = append(formulas, formula)
		}
	}
	// Reconstruct reader with this line and remaining content
	reader = io.MultiReader(
		strings.NewReader(commentBlock.String()),
		bufReader,
	)
	return processWithTBLFMLib(reader, inputFormat, writer, outputFormat, formulas)
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
	// Apply formulas
	if table, err = tblfm.Apply(table, formulas, tblfm.WithHeader(true)); err != nil {
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
