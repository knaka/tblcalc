package mlr

import (
	"io"

	"github.com/johnkerl/miller/v6/pkg/climain"
	"github.com/johnkerl/miller/v6/pkg/stream"
)

// Put runs Miller with the specified file and scripts.
// hasHeader indicates whether the first row should be treated as a header.
func Put(
	files []string,
	scripts []string,
	hasHeader bool,
	inputFormat string,
	outputFormat string,
	writeCloser io.WriteCloser,
) (
	err error,
) {
	args := []string{
		"mlr",
		// List of command-line flags - Miller Documentation https://miller.readthedocs.io/en/latest/reference-main-flag-list/
		// File formats - Miller Documentation https://miller.readthedocs.io/en/latest/file-formats/
		"--i" + inputFormat,
		"--o" + outputFormat,
		"--pass-comments",
	}
	if !hasHeader {
		args = append(args, "--implicit-csv-header")
	}
	args = append(args,
		// In-place mode - Miller Documentation https://miller.readthedocs.io/en/latest/reference-main-in-place-processing/
		// Windows tries to rename across drives.
		// "-I",
		// List of verbs - Miller Documentation https://miller.readthedocs.io/en/latest/reference-verbs/#put
		"put",
	)
	for _, script := range scripts {
		args = append(args, "-e", script)
	}
	options, recordTransformers, err := climain.ParseCommandLine(args)
	if err != nil {
		return
	}
	err = stream.Stream(files, options, recordTransformers, writeCloser, false)
	return
}
