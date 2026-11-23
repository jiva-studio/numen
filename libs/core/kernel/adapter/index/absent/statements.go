package absent

import _ "embed"

//go:embed sql/put.sql
var putSQL string

//go:embed sql/list.sql
var listSQL string

//go:embed sql/get.sql
var getSQL string

//go:embed sql/remove.sql
var removeSQL string
