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
		var record []string
		// Parse CSV or TSV line
		switch inputFormat {
		case InputFormatCSV:
			csvReader := csv.NewReader(strings.NewReader(line))
			record, err = csvReader.Read()
			if err != nil {
				return fmt.Errorf("failed to parse CSV line %d: %v", lineNum, err)
			}
		case InputFormatTSV:
			record = strings.Split(line, "\t")
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
	switch outputFormat {
	case OutputFormatCSV:
		return writeCSV(writer, table, commentLines, lineNum)
	case OutputFormatTSV:
		return writeTSV(writer, table, commentLines, lineNum)
	}
	return
}

func writeCSV(writer io.Writer, table [][]string, commentLines map[int]string, lineNum int) error {
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()
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

func writeTSV(writer io.Writer, table [][]string, commentLines map[int]string, lineNum int) error {
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
