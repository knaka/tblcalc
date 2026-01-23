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

	//lint:ignore ST1001
	//revive:disable-next-line:dot-imports
	//nolint:staticcheck
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

	inPlace               bool
	optForcedInputFormat  *tblcalc.InputFormat
	optForcedOutputFormat *tblcalc.OutputFormat
}

// stdinFileName is a special name for standard input.
const stdinFileName = "-"

// tblcalcEntry is the entry point.
func tblcalcEntry(params *tblcalcParams) (err error) {
	if params.verbose {
		for i, arg := range params.args {
			Must(fmt.Fprintln(params.stderr, "b5747de", i, arg))
		}
	}
	if len(params.args) == 0 {
		params.args = append(params.args, stdinFileName)
	}
	for _, inPath := range params.args {
		err = func() error {
			var inputFormat tblcalc.InputFormat
			var reader io.Reader
			if inPath == stdinFileName {
				if params.inPlace {
					return fmt.Errorf("cannot use in-place mode with standard input")
				}
				if params.optForcedInputFormat == nil {
					return fmt.Errorf("must specify input format with standard input")
				}
				inputFormat = *params.optForcedInputFormat
				reader = params.stdin
			} else {
				if params.optForcedInputFormat != nil {
					inputFormat = *params.optForcedInputFormat
				} else {
					ext := strings.ToLower(path.Ext(inPath))
					switch ext {
					case ".csv":
						inputFormat = tblcalc.InputFormatCSV
					case ".tsv":
						inputFormat = tblcalc.InputFormatTSV
					default:
						return fmt.Errorf("unexpected file extension \"%s\"", ext)
					}
				}
				inFile, err2 := os.Open(inPath)
				if err2 != nil {
					return fmt.Errorf("failed to open input file: %s Error: %v", inPath, err2)
				}
				defer (func() { Ignore(inFile.Close()) })()
				reader = inFile
			}
			var outputFormat tblcalc.OutputFormat
			if params.optForcedOutputFormat != nil {
				outputFormat = *params.optForcedOutputFormat
			} else {
				switch inputFormat {
				case tblcalc.InputFormatCSV:
					outputFormat = tblcalc.OutputFormatCSV
				case tblcalc.InputFormatTSV:
					outputFormat = tblcalc.OutputFormatTSV
				}
			}
			var optOutFile *os.File
			var writer io.Writer
			if params.inPlace {
				optOutFile, err = os.CreateTemp("", appID)
				if err != nil {
					return fmt.Errorf("failed to create temporary output file: %v", err)
				}
				defer func() {
					Ignore(optOutFile.Close())
					Must(os.Remove(optOutFile.Name()))
				}()
				writer = optOutFile
			} else {
				optOutFile = nil
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
				return fmt.Errorf("failed to preprocess: %v", err)
			}
			err = bufOut.Flush()
			if err != nil {
				return fmt.Errorf("failed to flush output: %v", err)
			}
			if params.inPlace {
				Must(optOutFile.Close())
				// Compare the original file with the output file
				var outContent []byte
				outContent, err = os.ReadFile(optOutFile.Name())
				if err != nil {
					return fmt.Errorf("failed to read output file: %s", optOutFile.Name())
				}
				source := Value(os.ReadFile(inPath))
				if bytes.Equal(source, outContent) {
					return nil
				}
				// Replace the original file content while preserving hard links
				origFile, err2 := os.OpenFile(inPath, os.O_WRONLY|os.O_TRUNC, 0)
				if err2 != nil {
					return fmt.Errorf("failed to open original file for writing: %s Error: %v", inPath, err2)
				}
				defer Must(origFile.Close())
				Must(origFile.Write(outContent))
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
		exeName: appID,
		stdin:   os.Stdin,
		stdout:  os.Stdout,
		stderr:  os.Stderr,
		isTerm:  term.IsTerminal(int(os.Stdout.Fd())),
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
		params.optForcedInputFormat = Ptr(tblcalc.InputFormatCSV)
	}
	if inputTSVForced {
		params.optForcedInputFormat = Ptr(tblcalc.InputFormatTSV)
	}
	if outputCSVForced {
		params.optForcedOutputFormat = Ptr(tblcalc.OutputFormatCSV)
	}
	if outputTSVForced {
		params.optForcedOutputFormat = Ptr(tblcalc.OutputFormatTSV)
	}
	err := tblcalcEntry(&params)
	if err != nil {
		log.Fatalf("%s: %v\n", appID, err)
	}
}
