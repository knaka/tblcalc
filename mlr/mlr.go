package mlr

import (
	"os"

	mlrentry "github.com/johnkerl/miller/v6/pkg/entrypoint"
)

// InplacePutMarkdown runs Miller with the specified file path and script for processing.
// filePath is the path to the input file. Miller, as a library, does not support processing data in memory.
func InplacePutMarkdown(filePath string, script string) {
	argsSave := os.Args
	defer func() { os.Args = argsSave }()
	os.Args = []string{
		"mlr",
		// List of command-line flags - Miller Documentation https://miller.readthedocs.io/en/latest/reference-main-flag-list/
		// File formats - Miller Documentation https://miller.readthedocs.io/en/latest/file-formats/
		"--imarkdown",
		"--omarkdown",
		// In-place mode - Miller Documentation https://miller.readthedocs.io/en/latest/reference-main-in-place-processing/
		"-I",
		// List of verbs - Miller Documentation https://miller.readthedocs.io/en/latest/reference-verbs/#put
		"put",
		"-e", script,
		filePath,
	}
	mlrentry.Main()
}

// InplacePut runs Miller with the specified file and scripts.
// filePath is the path to the input file. Miller, as a library, does not support processing data in memory.
// hasHeader indicates whether the first row should be treated as a header.
func InplacePut(filePath string, scripts []string, hasHeader bool, inputFormat, outputFormat string) {
	argsSave := os.Args
	defer func() { os.Args = argsSave }()
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
		"-I",
		// List of verbs - Miller Documentation https://miller.readthedocs.io/en/latest/reference-verbs/#put
		"put",
	)
	for _, script := range scripts {
		args = append(args, "-e", script)
	}
	args = append(args,
		filePath,
	)
	os.Args = args
	mlrentry.Main()
}
