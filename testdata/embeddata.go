package testdata

import _ "embed"

//go:embed test1.csv
var Test1CSV string

//go:embed test1-result.csv
var Test1ResultCSV string

//go:embed test1.tsv
var Test1TSV string

//go:embed test1-result.tsv
var Test1ResultTSV string

//go:embed test2.csv
var Test2CSV string

//go:embed test2-result.csv
var Test2ResultCSV string

//go:embed test3.csv
var Test3CSV string

//go:embed test3-exited.csv
var Test3ExitedCSV string

//go:embed test3-not-exited.csv
var Test3NotExitedCSV string
