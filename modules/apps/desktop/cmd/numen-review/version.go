package main

import "fmt"

// The tag this build was made from, how many commits stood behind it, and the
// commit itself. The linker writes all three in, and what stands here is what a
// build made by hand carries.
//
// The editor and this are always the same build: they ship together, one
// version, one installer.
var (
	version = "0.0.0-dev"
	build   = "0"
	commit  = "unknown"
)

func built() string {
	return fmt.Sprintf("numen-review %s (build %s, commit %s)", version, build, commit)
}
