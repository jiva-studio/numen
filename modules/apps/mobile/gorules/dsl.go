//go:build ruleguard

// Package gorules holds this module's require on the ruleguard DSL.
//
// The rules gocritic runs live in `modules/tools/gorules/timeeq.go`,
// outside every module, and they are written against this package.
// ruleguard typechecks them against the module it is linting, so each module
// has to carry the require. The tag keeps the file out of every build; a plain
// `//go:build ignore` would be dropped by `go mod tidy` and the linter would
// stop starting.
package gorules

import _ "github.com/quasilyte/go-ruleguard/dsl"
