package main

import "fmt"

// The tag this build was made from, how many commits stood behind it, and the
// commit itself. The linker writes all three in, and what stands here is what
// a build made by hand carries.
var (
	version = "0.0.0-dev"
	build   = "0"
	commit  = "unknown"
)

// built is what this binary calls itself: the version a person reads, the
// number that tells apart two builds of one version, and the commit that names
// the source it was made from.
func built() string {
	return fmt.Sprintf("numen %s (build %s, commit %s)", version, build, commit)
}
