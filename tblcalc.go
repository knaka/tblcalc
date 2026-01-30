// Package tblcalc provides table calculation.
package tblcalc

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"iter"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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
	formulas   []string
	scripts    []string
}

// Options is a functional options type.
type Options []funcopt.Option[tblcalcParams]

var WithIgnoreExit = funcopt.New(func(params *tblcalcParams, ignoreExit bool) {
	params.ignoreExit = ignoreExit
})

var WithFormulas = funcopt.New(func(params *tblcalcParams, formulas []string) {
	params.formulas = append(params.formulas, formulas...)
})

var WithScripts = funcopt.New(func(params *tblcalcParams, scripts []string) {
	params.scripts = append(params.scripts, scripts...)
})

// process is an internal function that handles both file and stream processing.
// If nullableReader is nil, it reads from filepath; otherwise it reads from the reader.
func process(
	filepath string,
	nullableReader io.Reader,
	inputFormat InputFormat,
	writer io.Writer,
	outputFormat OutputFormat,
	opts ...funcopt.Option[tblcalcParams],
) (
	err error,
) {
	params := tblcalcParams{}
	err = funcopt.Apply(&params, opts)
	if err != nil {
		return
	}
	reader := nullableReader
	if reader == nil {
		inFile, err2 := os.Open(filepath)
		if err2 != nil {
			return fmt.Errorf("failed to open input file: %s Error: %v", filepath, err2)
		}
		defer (func() { Must(inFile.Close()) })()
		reader = inFile
	}
	formulas := params.formulas
	scripts := params.scripts
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
	// Reconstruct reader with comment block and remaining content
	reader = io.MultiReader(
		strings.NewReader(commentBlock.String()),
		bufReader,
	)
	if len(formulas) > 0 {
		return processWithTBLFMLib(reader, inputFormat, writer, outputFormat, formulas, params.ignoreExit)
	} else if len(scripts) > 0 {
		// If input-stream is passed, write it to temporary file and pass to library function.
		if nullableReader != nil {
			inFile, err := os.CreateTemp("", "tblcalc-*")
			if err != nil {
				return fmt.Errorf("failed to create temp file: %w", err)
			}
			defer (func() {
				Ignore(inFile.Close())
				Must(os.Remove(inFile.Name()))
			})()
			Must(io.Copy(inFile, reader))
			Must(inFile.Close())
			filepath = inFile.Name()
		}
		return processWithMlr(filepath, inputFormat, writer, outputFormat, scripts, params.ignoreExit)
	}
	return
}

// ProcessStream reads data from reader, applies table formulas found in comment lines,
// and writes the result to writer. Comment lines starting with "# +TBLFM:" contain
// formulas that are applied to the table data. The input and output formats are
// specified by inputFormat and outputFormat parameters.
func ProcessStream(
	reader io.Reader,
	inputFormat InputFormat,
	writer io.Writer,
	outputFormat OutputFormat,
	opts ...funcopt.Option[tblcalcParams],
) error {
	return process("", reader, inputFormat, writer, outputFormat, opts...)
}

// ProcessFile reads data from filepath, applies table formulas found in comment lines,
// and writes the result to writer. Comment lines starting with "# +TBLFM:" contain
// formulas that are applied to the table data. The input and output formats are
// specified by inputFormat and outputFormat parameters.
// External files with extensions .skip, .tblfm, .mlr are searched in the same directory.
// The "%" character in these filenames acts as a wildcard (like SQL LIKE).
// For example, "foo%baz.csv.skip" matches "foo-bar-baz.csv".
// If a matching .skip file exists, no formulas or scripts are applied.
// If a matching .tblfm file exists, its contents are parsed as formulas
// (split by newlines and "::"). Similarly, if a matching .mlr file exists,
// its contents are used as a Miller script.
func ProcessFile(
	filePath string,
	inputFormat InputFormat,
	writer io.Writer,
	outputFormat OutputFormat,
	opts ...funcopt.Option[tblcalcParams],
) (
	err error,
) {
	dir := filepath.Dir(filePath)
	base := filepath.Base(filePath)
	// Check for .skip file
	if len(findMatchingFiles(dir, base, ".skip")) > 0 {
		if reader, err := os.Open(filePath); err != nil {
			return err
		} else {
			defer (func() { Must(reader.Close()) })()
			Must(io.Copy(writer, reader))
			return nil
		}
	}
	// Load formulas from matching .tblfm file
	var formulas []string
	for _, tblfmFile := range findMatchingFiles(dir, base, ".tblfm") {
		if content, err := os.ReadFile(tblfmFile); err == nil {
			formulas = append(formulas, splitFormulas(string(content))...)
		}
	}
	if len(formulas) > 0 {
		opts = append(opts, WithFormulas(formulas))
	}
	// Load script from matching .mlr file
	var scripts []string
	for _, mlrFile := range findMatchingFiles(dir, base, ".mlr") {
		if content, err := os.ReadFile(mlrFile); err == nil {
			script := strings.TrimSpace(string(content))
			if script != "" {
				scripts = append(scripts, script)
			}
		}
	}
	if len(scripts) > 0 {
		opts = append(opts, WithScripts(scripts))
	}
	return process(filePath, nil, inputFormat, writer, outputFormat, opts...)
}

// findMatchingFiles searches for files in dir that match the target filename
// with the given suffix. First checks for an exact match (target + suffix),
// then searches for wildcard patterns using "%" as the wildcard character.
func findMatchingFiles(dir, target, suffix string) []string {
	found := make(map[string]struct{})

	// First, check for exact match
	exactPath := filepath.Join(dir, target+suffix)
	if _, err := os.Stat(exactPath); err == nil {
		found[exactPath] = struct{}{}
	}

	// Search for wildcard patterns
	pattern := filepath.Join(dir, "*"+suffix)
	matches, err := filepath.Glob(pattern)
	if err == nil {
		for _, match := range matches {
			base := filepath.Base(match)
			patternBase := strings.TrimSuffix(base, suffix)
			if matchWildcard(patternBase, target) {
				found[match] = struct{}{}
			}
		}
	}

	if len(found) == 0 {
		return nil
	}

	result := make([]string, 0, len(found))
	for f := range found {
		result = append(result, f)
	}
	sort.Strings(result)
	return result
}

// matchWildcard checks if target matches the pattern where "%" acts as a wildcard.
// For example, "foo%baz" matches "foo-bar-baz" or "fooXYZbaz".
func matchWildcard(pattern, target string) bool {
	// If no wildcard, require exact match
	if !strings.Contains(pattern, "%") {
		return pattern == target
	}
	// Convert "%" to ".*" for regex matching
	regexPattern := "^" + regexp.QuoteMeta(pattern) + "$"
	regexPattern = strings.ReplaceAll(regexPattern, "%", ".*")
	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return false
	}
	return re.MatchString(target)
}

// splitFormulas splits content by newlines and "::" separator.
func splitFormulas(content string) []string {
	var result []string
	for line := range strings.SplitSeq(content, "\n") {
		for part := range strings.SplitSeq(line, "::") {
			part = strings.TrimSpace(part)
			if part != "" {
				result = append(result, part)
			}
		}
	}
	return result
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
	inPath string,
	inputFormat InputFormat,
	writer io.Writer,
	outputFormat OutputFormat,
	scripts []string,
	ignoreExit bool,
) (
	err error,
) {
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
		err = mlr.Put([]string{inPath}, mlrScripts, true, inFmt, outFmt, resultFile)
		if err != nil {
			return
		}
	}
	Must(resultFile.Close())
	resultFile = Value(os.Open(resultFile.Name()))
	if _, err = io.Copy(writer, resultFile); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}
	return
}
