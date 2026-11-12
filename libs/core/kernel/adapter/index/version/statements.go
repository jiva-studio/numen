package version

import _ "embed"

// The statements the package reads through a pool of the index database,
// embedded when the package is built.

//go:embed sql/get_path.sql
var getPathSQL string

//go:embed sql/list.sql
var listSQL string
