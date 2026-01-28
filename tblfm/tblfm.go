// Package tblfm handles table formulas using Lua for expression evaluation.
package tblfm

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	lua "github.com/yuin/gopher-lua"
)

// Option is a functional option for Apply.
type Option func(*config)

// Options is a functional options for Apply.
type Options []func(*config)

// config holds the configuration for Apply.
type config struct {
	hasHeader  bool
	ignoreExit bool
}

// WithHeader specifies whether the first row is a header row.
// Default is true (has header).
func WithHeader(hasHeader bool) Option {
	return func(c *config) {
		c.hasHeader = hasHeader
	}
}

// WithHeader specifies whether the first row is a header row.
// Default is true (has header).
func WithIgnoreExit(ignoreExit bool) Option {
	return func(c *config) {
		c.ignoreExit = ignoreExit
	}
}

// Base patterns for row/column specifications.
// These are used to build more complex regular expressions.
const (
	// specValPat matches the value part of a row/column specification:
	// - Absolute position: 1, 2, 3, ...
	// - Relative position: -1, +2, ...
	// - Special markers: <, <<, <<<, >, >>, >>>
	// - Header name reference: {header name} (for columns only, when hasHeader is true)
	specValPat = `[-+]?\d+|<{1,3}|>{1,3}|\{[^}]+\}`

	// rowSpecPat matches a row specification like @2, @-1, @<, @>>
	rowSpecPat = `@(` + specValPat + `)`

	// colSpecPat matches a column specification like $3, $-1, $<, $>>, ${header name}
	colSpecPat = `\$(` + specValPat + `)`

	// cellSpecPat matches a cell specification (optional row + optional column)
	// Examples: @2$3, $4, @3, @<$>, ${Price}
	cellSpecPat = `(?:@(?:` + specValPat + `))?(?:\$(?:` + specValPat + `))?`
)

// regexps holds all compiled regular expressions used for TBLFM parsing.
type regexps struct {
	formula             *regexp.Regexp
	formulaStartPosSpec int // Capture group index for start position spec (e.g., "@2$3")
	formulaEndPosSpec   int // Capture group index for end position spec (e.g., "@>>$>")
	formulaExpression   int // Capture group index for expression (e.g., "$2*$3")

	cellRef        *regexp.Regexp
	cellRefRowSpec int // Capture group index for row spec value (e.g., "2" from "@2")
	cellRefColSpec int // Capture group index for col spec value (e.g., "3" from "$3")

	rowRef        *regexp.Regexp
	rowRefRowSpec int // Capture group index for row spec value

	cellPos        *regexp.Regexp
	cellPosRowSpec int // Capture group index for row spec value
	cellPosColSpec int // Capture group index for col spec value

	rangeRef         *regexp.Regexp
	rangeRefStartPos int // Capture group index for start position
	rangeRefEndPos   int // Capture group index for end position
}

// getRegexps returns all compiled regular expressions.
// Uses sync.OnceValue to ensure regexp.MustCompile is only called once.
var getRegexps = sync.OnceValue(func() *regexps {
	return &regexps{
		// Formula parser: supports $4=$2*$3 (column), @3=@2 (row), @3$4=@2$2 (cell)
		// Also supports range syntax: @2$>..@>>$>=@1$>
		formula:             regexp.MustCompile(`^(` + cellSpecPat + `)(?:\.\.(` + cellSpecPat + `))?\s*=\s*(.+)$`),
		formulaStartPosSpec: 1,
		formulaEndPosSpec:   2,
		formulaExpression:   3,

		// Find cell references like @2$3, $2, $3, $-1, $-2 (with optional row)
		// Supports <, <<, <<< (up to 3 levels) and >, >>, >>> (up to 3 levels)
		// Capture groups: 1=@row (optional), 2=row value, 3=$col, 4=col value
		cellRef:        regexp.MustCompile(`(` + rowSpecPat + `)?(` + colSpecPat + `)`),
		cellRefRowSpec: 2,
		cellRefColSpec: 4,

		// Find standalone row references like @2, @<, @<<, @<<< (this will also match @2$ but we process cellRefRe first)
		rowRef:        regexp.MustCompile(rowSpecPat),
		rowRefRowSpec: 1,

		// Parse cell position like @2$3, $4, @3
		cellPos:        regexp.MustCompile(`^(?:` + rowSpecPat + `)?(?:` + colSpecPat + `)?$`),
		cellPosRowSpec: 1,
		cellPosColSpec: 2,

		// Find range references like @<..@>> or @2$1..@5$3
		rangeRef:         regexp.MustCompile(`(` + cellSpecPat + `)\.\.(` + cellSpecPat + `)`),
		rangeRefStartPos: 1,
		rangeRefEndPos:   2,
	}
})

// resolveColSpec resolves a column specification to a 0-based column index.
// colSpec can be: numeric (1-based), relative (-1), special (<, >, etc.), or header name ({name}).
// Returns (-1, nil) if not specified, or (index, nil) on success, or (-1, error) on failure.
func resolveColSpec(colSpec string, rowLen int, currentCol int, headerColMap map[string]int) (int, error) {
	if colSpec == "" {
		return -1, nil
	}

	switch colSpec {
	case "<":
		return 0, nil
	case "<<":
		return 1, nil
	case "<<<":
		return 2, nil
	case ">":
		return rowLen - 1, nil
	case ">>":
		return rowLen - 2, nil
	case ">>>":
		return rowLen - 3, nil
	default:
		// Check for header name reference: {header name}
		if strings.HasPrefix(colSpec, "{") && strings.HasSuffix(colSpec, "}") {
			headerName := colSpec[1 : len(colSpec)-1]
			if colIdx, ok := headerColMap[headerName]; ok {
				return colIdx, nil
			}
			return -1, fmt.Errorf("header column %q not found (hasHeader may be false or header name is incorrect)", headerName)
		}

		// Numeric column reference
		colNum, _ := strconv.Atoi(colSpec)
		if colNum > 0 {
			colIdx := colNum - 1 // 1-based to 0-based
			if rowLen > 0 && colIdx >= rowLen {
				return -1, fmt.Errorf("column index $%d is out of range (max columns: %d)", colNum, rowLen)
			}
			return colIdx, nil
		} else if colNum < 0 && currentCol > 0 {
			// Relative reference: $-1 means one column to the left
			colIdx := currentCol - 1 + colNum
			if colIdx < 0 {
				return -1, fmt.Errorf("relative column reference $%d results in negative index", colNum)
			}
			return colIdx, nil
		}
	}
	return -1, nil
}

// parseCellPosition parses a cell position specification like "@2$3", "$4", "@3", "${Price}"
// Returns (row, col, err) where -1 means "any" (not specified)
// currentRow and currentCol are 1-based positions used for relative references
// headerColMap maps header names to 0-based column indices (for ${header name} syntax)
func parseCellPosition(pos string, tableLen int, rowLen int, currentRow int, currentCol int, headerColMap map[string]int) (row int, col int, err error) {
	row = -1
	col = -1

	if pos == "" {
		return
	}

	re := getRegexps()
	matches := re.cellPos.FindStringSubmatch(pos)
	if matches == nil {
		return
	}

	rowSpec := matches[re.cellPosRowSpec]
	colSpec := matches[re.cellPosColSpec]

	// Parse row
	if rowSpec != "" {
		switch rowSpec {
		case "<":
			row = 0 // First row (header if exists)
		case "<<":
			row = 1 // Second row
		case "<<<":
			row = 2 // Third row
		case ">":
			row = tableLen - 1
		case ">>":
			row = tableLen - 2
		case ">>>":
			row = tableLen - 3
		default:
			rowNum, _ := strconv.Atoi(rowSpec)
			if rowNum > 0 {
				row = rowNum - 1 // 1-based to 0-based
			} else if rowNum < 0 && currentRow > 0 {
				// Relative reference: @-1 means one row above current
				row = currentRow - 1 + rowNum
			}
		}
	}

	// Parse column using shared resolver
	col, err = resolveColSpec(colSpec, rowLen, currentCol, headerColMap)

	return
}

// Apply performs table calculations using TBLFM formulas on the input 2D array and returns the modified table.
func Apply(
	table [][]string, // Input table (modified in place)
	formulas []string, // TBLFM formula strings
	opts ...Option, // Functional options
) (
	resultTable [][]string, // Updated table (or the same pointer)
	err error,
) {
	cfg := &config{
		hasHeader: true, // Default: has header
	}
	for _, opt := range opts {
		opt(cfg)
	}

	resultTable = table

	// If formulas are empty, do nothing
	if len(formulas) == 0 {
		return
	}

	// Determine data row start position
	dataStartRow := 0
	if cfg.hasHeader {
		dataStartRow = 1
	}

	// Build header column map for ${header name} references
	headerColMap := make(map[string]int)
	if cfg.hasHeader && len(table) > 0 {
		for colIdx, headerName := range table[0] {
			headerColMap[headerName] = colIdx
		}
	}

	// Create Lua state
	L := lua.NewState()
	defer L.Close()

	// Register built-in functions
	registerBuiltinFunctions(L)

	// Apply each formula in order
	for _, formula := range formulas {
		formula = strings.TrimSpace(formula)
		if formula == "" {
			continue
		}

		if formula == "exit" {
			if cfg.ignoreExit {
				continue
			} else {
				return
			}
		}

		// Parse formula
		re := getRegexps()
		matches := re.formula.FindStringSubmatch(formula)
		if matches == nil {
			return resultTable, fmt.Errorf("invalid formula format: %s", formula)
		}

		// e.g., "@2$>" or "$4" or empty
		startPosSpec := matches[re.formulaStartPosSpec]
		endPosSpec := matches[re.formulaEndPosSpec] // e.g., "@>>$>" or empty (if no range)
		expression := matches[re.formulaExpression]

		// Determine maximum row length for column parsing
		maxRowLen := 0
		for _, r := range table {
			if len(r) > maxRowLen {
				maxRowLen = len(r)
			}
		}

		// Parse start position (no current position for target specification)
		targetStartRow, targetStartCol, err := parseCellPosition(startPosSpec, len(table), maxRowLen, 0, 0, headerColMap)
		if err != nil {
			return resultTable, fmt.Errorf("invalid target position %q: %w", startPosSpec, err)
		}

		// Parse end position (if range specified)
		var targetEndRow, targetEndCol = -1, -1
		if endPosSpec != "" {
			targetEndRow, targetEndCol, err = parseCellPosition(endPosSpec, len(table), maxRowLen, 0, 0, headerColMap)
			if err != nil {
				return resultTable, fmt.Errorf("invalid target end position %q: %w", endPosSpec, err)
			}
		}

		// Determine target range
		var targetRowStart, targetRowEnd int
		var targetColStart, targetColEnd int

		if endPosSpec == "" {
			// Single cell or column/row specification
			targetRowStart = targetStartRow
			targetRowEnd = targetStartRow
			targetColStart = targetStartCol
			targetColEnd = targetStartCol
		} else {
			// Range specification
			targetRowStart = targetStartRow
			targetRowEnd = targetEndRow
			targetColStart = targetStartCol
			targetColEnd = targetEndCol
		}

		// Double loop: iterate over all rows and columns
		for rowIdx := dataStartRow; rowIdx < len(table); rowIdx++ {
			row := table[rowIdx]

			// Check if this row matches the target range
			if targetRowStart != -1 && rowIdx < targetRowStart {
				continue // Skip rows before start
			}
			if targetRowEnd != -1 && rowIdx > targetRowEnd {
				continue // Skip rows after end
			}

			for colIdx := 0; colIdx < len(row); colIdx++ {
				// Check if this column matches the target range
				if targetColStart != -1 && colIdx < targetColStart {
					continue // Skip columns before start
				}
				if targetColEnd != -1 && colIdx > targetColEnd {
					continue // Skip columns after end
				}

				// This cell is a target, evaluate the expression
				currentRow := rowIdx + 1 // 1-based
				currentCol := colIdx + 1 // 1-based

				// Evaluate the expression using Lua
				resultStr, err := evaluateExpression(L, expression, table, currentRow, currentCol, dataStartRow, headerColMap)
				if err != nil {
					return resultTable, fmt.Errorf("error evaluating formula %s at @%d$%d: %w", formula, currentRow, currentCol, err)
				}

				// Set result to target cell
				table[rowIdx][colIdx] = resultStr
			}
		}
	}

	return resultTable, nil
}

// evaluateExpression evaluates a Lua expression with cell references replaced by actual values
func evaluateExpression(L *lua.LState, expression string, table [][]string, currentRow int, currentCol int, dataStartRow int, headerColMap map[string]int) (string, error) {
	// Replace cell and row references with Lua code
	evaluableExpr := expression

	// First, replace range references with Lua table literals
	re := getRegexps()
	var replaceErr error
	evaluableExpr = re.rangeRef.ReplaceAllStringFunc(evaluableExpr, func(rangeRef string) string {
		if replaceErr != nil {
			return rangeRef // Skip processing if error already occurred
		}

		matches := re.rangeRef.FindStringSubmatch(rangeRef)
		if matches == nil {
			return rangeRef
		}

		startPos := matches[re.rangeRefStartPos]
		endPos := matches[re.rangeRefEndPos]

		if startPos == "" || endPos == "" {
			return rangeRef
		}

		// Expand the range into a Lua table
		values, err := expandRange(startPos, endPos, table, currentRow, currentCol, dataStartRow, headerColMap)
		if err != nil {
			replaceErr = err
			return rangeRef
		}

		// Convert to Lua table literal string
		var parts []string
		for _, v := range values {
			switch val := v.(type) {
			case float64:
				parts = append(parts, strconv.FormatFloat(val, 'f', -1, 64))
			case string:
				// Skip empty strings in numeric contexts
				if val != "" {
					// Quote strings
					parts = append(parts, fmt.Sprintf("%q", val))
				}
			default:
				parts = append(parts, fmt.Sprintf("%v", val))
			}
		}

		return "{" + strings.Join(parts, ",") + "}"
	})
	if replaceErr != nil {
		return "", replaceErr
	}

	// Then, replace cell references (with optional row) like @2$3, $2, ${Price}
	evaluableExpr = re.cellRef.ReplaceAllStringFunc(evaluableExpr, func(ref string) string {
		if replaceErr != nil {
			return ref // Skip processing if error already occurred
		}

		matches := re.cellRef.FindStringSubmatch(ref)
		if matches == nil {
			return ref
		}

		rowSpec := matches[re.cellRefRowSpec]
		colSpec := matches[re.cellRefColSpec]

		// Determine source row
		var sourceRow int
		if rowSpec == "" {
			sourceRow = currentRow - 1 // 1-based to 0-based
		} else {
			switch rowSpec {
			case "<":
				sourceRow = 0
			case "<<":
				sourceRow = 1
			case "<<<":
				sourceRow = 2
			case ">":
				sourceRow = len(table) - 1
			case ">>":
				sourceRow = len(table) - 2
			case ">>>":
				sourceRow = len(table) - 3
			default:
				rowNum, _ := strconv.Atoi(rowSpec)
				if rowNum < 0 {
					sourceRow = currentRow - 1 + rowNum
				} else {
					sourceRow = rowNum - 1
				}
			}
		}

		// Determine source column using shared resolver
		rowLen := 0
		if sourceRow >= 0 && sourceRow < len(table) {
			rowLen = len(table[sourceRow])
		}
		sourceCol, err := resolveColSpec(colSpec, rowLen, currentCol, headerColMap)
		if err != nil {
			replaceErr = err
			return ref
		}

		// Get the cell value
		if sourceRow >= 0 && sourceRow < len(table) &&
			sourceCol >= 0 && sourceCol < len(table[sourceRow]) {
			cellValue := table[sourceRow][sourceCol]

			// Try to parse as number
			if num, err := strconv.ParseFloat(cellValue, 64); err == nil {
				return strconv.FormatFloat(num, 'f', -1, 64)
			}

			// Return as quoted string
			return fmt.Sprintf("%q", cellValue)
		}
		return "0"
	})
	if replaceErr != nil {
		return "", replaceErr
	}

	// Then, replace standalone row references like @<, @<<, @> (for row copy operations)
	evaluableExpr = re.rowRef.ReplaceAllStringFunc(evaluableExpr, func(ref string) string {
		matches := re.rowRef.FindStringSubmatch(ref)
		if matches == nil {
			return ref
		}

		rowSpec := matches[re.rowRefRowSpec]
		var sourceRow int

		switch rowSpec {
		case "<":
			sourceRow = 0
		case "<<":
			sourceRow = 1
		case "<<<":
			sourceRow = 2
		case ">":
			sourceRow = len(table) - 1
		case ">>":
			sourceRow = len(table) - 2
		case ">>>":
			sourceRow = len(table) - 3
		default:
			rowNum, _ := strconv.Atoi(rowSpec)
			if rowNum < 0 {
				sourceRow = currentRow - 1 + rowNum
			} else {
				sourceRow = rowNum - 1
			}
		}

		// For row copy operations, return the value from the same column in the source row
		if sourceRow >= 0 && sourceRow < len(table) &&
			currentCol-1 >= 0 && currentCol-1 < len(table[sourceRow]) {
			cellValue := table[sourceRow][currentCol-1]

			// Try to parse as number
			if num, err := strconv.ParseFloat(cellValue, 64); err == nil {
				return strconv.FormatFloat(num, 'f', -1, 64)
			}

			// Return as quoted string
			return fmt.Sprintf("%q", cellValue)
		}
		return "0"
	})

	// Execute Lua code to get result
	if err := L.DoString("return " + evaluableExpr); err != nil {
		return "", err
	}

	// Get the result from Lua stack
	ret := L.Get(-1)
	L.Pop(1)

	// Convert result to string
	switch v := ret.(type) {
	case *lua.LNumber:
		num := float64(*v)
		if num == float64(int64(num)) {
			return strconv.FormatInt(int64(num), 10), nil
		}
		return strconv.FormatFloat(num, 'f', -1, 64), nil
	case lua.LString:
		return string(v), nil
	case lua.LBool:
		if v {
			return "true", nil
		}
		return "false", nil
	default:
		return fmt.Sprintf("%v", ret), nil
	}
}

// expandRange expands a range reference like "@<..@>>" into an array of values
func expandRange(startPos, endPos string, table [][]string, currentRow, currentCol, dataStartRow int, headerColMap map[string]int) ([]any, error) {
	maxRowLen := 0
	for _, r := range table {
		if len(r) > maxRowLen {
			maxRowLen = len(r)
		}
	}

	startRow, startCol, err := parseCellPosition(startPos, len(table), maxRowLen, currentRow, currentCol, headerColMap)
	if err != nil {
		return nil, fmt.Errorf("invalid range start %q: %w", startPos, err)
	}
	endRow, endCol, err := parseCellPosition(endPos, len(table), maxRowLen, currentRow, currentCol, headerColMap)
	if err != nil {
		return nil, fmt.Errorf("invalid range end %q: %w", endPos, err)
	}

	var values []any

	// If only row is specified (no column), assume current column
	if startCol == -1 && endCol == -1 {
		startCol = currentCol - 1
		endCol = currentCol - 1
	}

	// If only column is specified (no row):
	// - If columns differ (horizontal range like $1..$4), use current row only
	// - If columns are same (vertical range like $1..$1), iterate through all data rows
	if startRow == -1 && endRow == -1 {
		if startCol != endCol {
			// Horizontal range: use current row
			startRow = currentRow - 1
			endRow = currentRow - 1
		} else {
			// Vertical range: iterate through all data rows
			startRow = dataStartRow
			endRow = len(table) - 1
		}
	}

	// Expand the range
	for r := startRow; r >= 0 && r <= endRow && r < len(table); r++ {
		for c := startCol; c >= 0 && c <= endCol && c < len(table[r]); c++ {
			val := table[r][c]
			// Try to parse as number
			if num, err := strconv.ParseFloat(val, 64); err == nil {
				values = append(values, num)
			} else {
				values = append(values, val)
			}
		}
	}

	return values, nil
}
