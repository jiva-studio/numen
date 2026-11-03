package searchpolicy

import _ "embed"

// The statements the package reads through a pool of the index database,
// embedded when the package is built.

//go:embed sql/passages.sql
var passagesSQL string

//go:embed sql/read.sql
var readSQL string
