package main

import "fmt"

// The tag this build was made from, and how many commits stood behind it. The
// linker writes both in, and what stands here is what a build made by hand
// carries.
var (
	version = "0.0.0-dev"
	build   = "0"
)

// built is what this binary calls itself: the version a person reads, and the
// number that tells apart two builds of one version.
func built() string {
	return fmt.Sprintf("numen %s (build %s)", version, build)
}
