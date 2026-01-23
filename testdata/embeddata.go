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
