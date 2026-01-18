// Package tblcalc provides table calculation.
package tblcalc

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"

	"github.com/knaka/tblcalc/tblfm"
)

// InputFormat represents the format of input data.
type InputFormat int

const (
	// InputFormatNone indicates no format is specified.
	InputFormatNone InputFormat = iota
	// InputFormatCSV indicates CSV (Comma-Separated Values) format.
	InputFormatCSV
	// InputFormatTSV indicates TSV (Tab-Separated Values) format.
	InputFormatTSV
)

// OutputFormat represents the format of output data.
type OutputFormat int

const (
	// OutputFormatNone indicates no format is specified.
	OutputFormatNone OutputFormat = iota
	// OutputFormatCSV indicates CSV (Comma-Separated Values) format.
	OutputFormatCSV
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
	_ = inputFormat
	_ = outputFormat

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

	// Read input line by line, preserving comments
	scanner := bufio.NewScanner(reader)
	lineNum := 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			commentLines[lineNum] = line
			lineNum++
			continue
		}

		// Parse CSV or TSV line
		var csvReader *csv.Reader
		switch inputFormat {
		case InputFormatCSV:
			csvReader = csv.NewReader(strings.NewReader(line))
		case InputFormatTSV:
			csvReader = csv.NewReader(strings.NewReader(line))
			csvReader.Comma = '\t'
		case InputFormatNone:
			panic("983d695")
		}

		record, err2 := csvReader.Read()
		if err2 != nil {
			return fmt.Errorf("failed to parse line %d: %v", lineNum, err2)
		}

		table = append(table, record)
		lineNum++
	}

	if err = scanner.Err(); err != nil {
		return err
	}

	// Apply formulas
	if table, err = tblfm.Apply(table, formulas, tblfm.WithHeader(true)); err != nil {
		return fmt.Errorf("failed to apply formulas: %v", err)
	}

	// Write output with comments preserved
	var csvWriter *csv.Writer
	switch outputFormat {
	case OutputFormatCSV:
		csvWriter = csv.NewWriter(writer)
	case OutputFormatTSV:
		csvWriter = csv.NewWriter(writer)
		csvWriter.Comma = '\t'
	case OutputFormatNone:
		panic("cbb4884")
	}

	tableLineNum := 0
	for i := 0; i < lineNum; i++ {
		if comment, isComment := commentLines[i]; isComment {
			// Write comment line
			if _, err = fmt.Fprintln(writer, comment); err != nil {
				return err
			}
		} else {
			// Write table row
			if tableLineNum < len(table) {
				if err = csvWriter.Write(table[tableLineNum]); err != nil {
					return err
				}
				tableLineNum++
			}
		}
	}

	csvWriter.Flush()
	if err = csvWriter.Error(); err != nil {
		return err
	}

	return nil
}
