# tblcalc(1) - Table Calculator

## Core Functionality

tblcalc is a table calculation tool that "applies formulas to CSV/TSV tables using special comment directives, similar to Emacs Org-mode's table formulas."

## Key Features

**Formula Processing:**

1. **+TBLFM Directive** - Applies table formulas specified in comment lines to CSV/TSV data files. Formulas use column references (e.g., `$2`, `$3`, `$4`) to perform calculations across table cells.

2. **Comment Preservation** - Maintains all comment lines (lines starting with `#`) in their original positions while processing table data.

3. **Format Support** - Supports both CSV (Comma-Separated Values) and TSV (Tab-Separated Values) formats for input and output.

4. **In-Place Editing** - Allows direct file modification with the `-i` flag, preserving hard links.

## Installation & Usage

Install via: `go install github.com/knaka/tblcalc/cmd/tblcalc@latest`

Basic commands:
- Process CSV file: `tblcalc input.csv >output.csv`
- In-place editing: `tblcalc -i file.csv`
- Force format: `tblcalc --icsv --ocsv file.txt`

### Example

Input file (test.csv):
```csv
# Product list
#
# +TBLFM: $4=$2*$3
#
"Product","Unit Price",Stock,Total
Apple,100,50,
Banana,80,30,
Orange,120,20,
```

After processing with `tblcalc test.csv`:
```csv
# Product list
#
# +TBLFM: $4=$2*$3
#
"Product","Unit Price",Stock,Total
Apple,100,50,5000
Banana,80,30,2400
Orange,120,20,2400
```

## Command-Line Options

- `-h, --help` - Show help message
- `-i, --in-place` - Edit file(s) in-place
- `-v, --verbose` - Enable verbose output
- `--icsv` - Force CSV for input format
- `--itsv` - Force TSV for input format
- `--ocsv` - Force CSV for output format
- `--otsv` - Force TSV for output format

## License

Released under the Apache 2.0 License.
