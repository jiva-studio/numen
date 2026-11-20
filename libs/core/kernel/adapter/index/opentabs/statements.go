package opentabs

import _ "embed"

//go:embed sql/list.sql
var listSQL string

//go:embed sql/save.sql
var saveSQL string
