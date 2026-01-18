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

## Editor Integration

### VSCode Configuration

You can configure VSCode to automatically apply formulas when saving CSV/TSV files using the [Run on Save](https://github.com/emeraldwalk/vscode-runonsave) extension.

Add this to your `.vscode/settings.json`:

```json
{
  "emeraldwalk.runonsave": {
    "commands": [
      {
        "match": "\\.(csv|tsv)$",
        "cmd": "tblcalc -i ${file}"
      }
    ]
  }
}
```

This automatically processes files with `+TBLFM` directives whenever you save them.

If you use the [Rainbow CSV](https://marketplace.visualstudio.com/items?itemName=mechatroner.rainbow-csv) extension for better CSV/TSV visualization, configure it to recognize comment lines:

```json
{
  "rainbow_csv.comment_prefix": "#"
}
```

This ensures that comment lines (including `+TBLFM` directives) are properly displayed and not treated as data rows.

## Command-Line Options

- `-h, --help` - Show help message
- `-i, --in-place` - Edit file(s) in-place
- `-v, --verbose` - Enable verbose output
- `--icsv` - Force CSV for input format
- `--itsv` - Force TSV for input format
- `--ocsv` - Force CSV for output format
- `--otsv` - Force TSV for output format

## Formula Syntax

### Cell Reference Notation

The `+TBLFM` directive uses Org-mode-style cell references:
- `$2`, `$3`, `$4` - Column references (1-indexed)
- `@2` - Row 2 (first data row after header)
- `$>` - Last column
- `@>` - Last row
- `@>>` - Second-to-last row
- `@<` - First row (including header)
- `@2$3` - Cell at row 2, column 3
- Ranges: `@<<$>..@>>$>` (range notation using `..`)

### Lua-Based Formulas

Formulas are evaluated using Lua, providing flexible syntax for:
- Arithmetic operations: `$2*$3`, `$2+$3-10`
- String operations: String concatenation and manipulation
- Conditional expressions: `$2 > 100 and $2*0.9 or $2`
- All standard Lua built-in functions and libraries

### Vector Functions

Available aggregation functions for ranges:
- `vsum(range)` - Sum of values
- `vmean(range)` - Average of values
- `vmedian(range)` - Median of values
- `vmax(range)` - Maximum value
- `vmin(range)` - Minimum value

Example: `vsum(@2$3..@>$3)` calculates the sum of column 3 from row 2 to the last row.

### Multiple Formulas

Multiple formulas can be specified with:
- Multiple `# +TBLFM:` lines
- Separated by `::` in a single line: `# +TBLFM: $4=$2*$3 :: $5=$2+$3`

### Important Note

Lua's string concatenation operator `..` visually resembles the range operator `..`. To prevent confusion:
- Add spaces around concatenation: `$2 .. " items"`
- Use parentheses around cell references: `($2)..($3)`

## License

Released under the Apache 2.0 License.
