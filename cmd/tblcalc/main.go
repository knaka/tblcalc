// Main package.
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"

	"github.com/knaka/tblcalc"
	"github.com/spf13/pflag"
	"golang.org/x/term"

	//revive:disable-next-line:dot-imports
	. "github.com/knaka/go-utils"
)

var appID = "tblcalc"

type tblcalcParams struct {
	exeName string
	stdin   io.Reader
	stdout  io.Writer
	stderr  io.Writer
	isTerm  bool
	args    []string

	verbose bool
	colored bool

	inPlace            bool
	forcedInputFormat  tblcalc.InputFormat
	forcedOutputFormat tblcalc.OutputFormat
}

// stdinFileName is a special name for standard input.
const stdinFileName = "-"

// tblcalcEntry is the entry point.
func tblcalcEntry(params *tblcalcParams) (err error) {
	if params.verbose {
		for i, arg := range params.args {
			log.Println(i, arg)
		}
	}
	for _, inPath := range params.args {
		err = func() error {
			inputFormat := params.forcedInputFormat
			outputFormat := params.forcedOutputFormat
			var inFile *os.File
			var reader io.Reader
			if inPath == stdinFileName {
				if params.inPlace {
					return fmt.Errorf("Cannot use in-place mode with standard input")
				}
				if inputFormat == tblcalc.InputFormatNone {
					return fmt.Errorf("Must specify input format with standard input")
				}
				reader = params.stdin
			} else {
				if inputFormat == tblcalc.InputFormatNone {
					ext := strings.ToLower(path.Ext(inPath))
					switch ext {
					case ".csv":
						inputFormat = tblcalc.InputFormatCSV
					case ".tsv":
						inputFormat = tblcalc.InputFormatTSV
					default:
						return fmt.Errorf("Unexpected file extension \"%s\"", ext)
					}
				}
				inFile, err = os.Open(inPath)
				if err != nil {
					return fmt.Errorf("Failed to open input file: %s Error: %v", inPath, err)
				}
				defer inFile.Close()
				reader = inFile
			}
			var outFile *os.File
			var writer io.Writer
			if params.inPlace {
				if outputFormat == tblcalc.OutputFormatNone {
					switch inputFormat {
					case tblcalc.InputFormatCSV:
						outputFormat = tblcalc.OutputFormatCSV
					case tblcalc.InputFormatTSV:
						outputFormat = tblcalc.OutputFormatTSV
					}
				}
				outFile, err = os.CreateTemp("", appID)
				if err != nil {
					return fmt.Errorf("Failed to create temporary output file: %v", err)
				}
				defer func() {
					outFile.Close()
					os.Remove(outFile.Name())
				}()
				writer = outFile
			} else {
				writer = params.stdout
			}
			bufOut := bufio.NewWriter(writer)
			err = tblcalc.Execute(
				reader,
				inputFormat,
				bufOut,
				outputFormat,
			)
			if err != nil {
				return fmt.Errorf("Failed to preprocess: %v", err)
			}
			err = bufOut.Flush()
			if err != nil {
				return fmt.Errorf("Failed to flush output: %v", err)
			}
			if params.inPlace {
				err = outFile.Close()
				if err != nil {
					return fmt.Errorf("Failed to close output file: %s Error: %v", outFile.Name(), err)
				}
				// Compare the original file with the output file
				var outContent []byte
				outContent, err = os.ReadFile(outFile.Name())
				if err != nil {
					return fmt.Errorf("Failed to read output file: %s", outFile.Name())
				}
				source := Value(os.ReadFile(inPath))
				if bytes.Equal(source, outContent) {
					return nil
				}
				// Replace the original file content while preserving hard links
				var origFile *os.File
				origFile, err = os.OpenFile(inPath, os.O_WRONLY|os.O_TRUNC, 0)
				if err != nil {
					return fmt.Errorf("Failed to open original file for writing: %s Error: %v", inPath, err)
				}
				defer origFile.Close()
				_, err = origFile.Write(outContent)
				if err != nil {
					return fmt.Errorf("Failed to write to original file: %s Error: %v", inPath, err)
				}
			}
			return nil
		}()
		if err != nil {
			break
		}
	}
	return err
}

func main() {
	params := tblcalcParams{
		exeName:            appID,
		stdin:              os.Stdin,
		stdout:             os.Stdout,
		stderr:             os.Stderr,
		forcedInputFormat:  tblcalc.InputFormatNone,
		forcedOutputFormat: tblcalc.OutputFormatNone,
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		params.stdin = bufio.NewReader(os.Stdin)
	}
	if term.IsTerminal(int(os.Stdout.Fd())) {
		params.isTerm = true
	} else {
		bufStdout := bufio.NewWriter(os.Stdout)
		defer bufStdout.Flush()
		params.stdout = bufStdout
	}
	var shouldPrintHelp bool
	pflag.BoolVarP(&shouldPrintHelp, "help", "h", false, "show help")

	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] [arg...]\n\nFlags:\n", appID)
		pflag.PrintDefaults()
	}
	pflag.BoolVarP(&params.verbose, "verbose", "v", false, "verbosity")
	pflag.BoolVarP(&params.colored, "colored", "c", params.isTerm, "colored")

	pflag.BoolVarP(&params.inPlace, "in-place", "i", false, "Edit file(s) in-place")

	var inputCSVForced bool
	pflag.BoolVarP(&inputCSVForced, "icsv", "", false, "Force CSV for input format")
	var inputTSVForced bool
	pflag.BoolVarP(&inputTSVForced, "itsv", "", false, "Force TSV for input format")

	var outputCSVForced bool
	pflag.BoolVarP(&outputCSVForced, "ocsv", "", false, "Force CSV for output format")
	var outputTSVForced bool
	pflag.BoolVarP(&outputTSVForced, "otsv", "", false, "Force TSV for output format")

	pflag.Parse()
	params.args = pflag.Args()
	if shouldPrintHelp {
		pflag.Usage()
		return
	}
	if inputCSVForced {
		params.forcedInputFormat = tblcalc.InputFormatCSV
	}
	if inputTSVForced {
		params.forcedInputFormat = tblcalc.InputFormatTSV
	}
	if outputCSVForced {
		params.forcedOutputFormat = tblcalc.OutputFormatCSV
	}
	if outputTSVForced {
		params.forcedOutputFormat = tblcalc.OutputFormatTSV
	}
	err := tblcalcEntry(&params)
	if err != nil {
		log.Fatalf("%s: %v\n", appID, err)
	}
}
