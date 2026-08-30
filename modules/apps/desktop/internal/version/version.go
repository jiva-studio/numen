// Package version is what a build of this application calls itself.
//
// The two windows ship together, one version, one installer, and the linker
// writes the numbers into this package for both of them.
package version

import "fmt"

// The tag this build was made from, how many commits stood behind it, and the
// commit itself. A build made by hand carries what stands here.
var (
	version = "0.0.0-dev"
	build   = "0"
	commit  = "unknown"
)

// Built is what the binary of this name says when it is asked: the version a
// person reads, the number that tells apart two builds of one version, and the
// commit that names the source it was made from.
func Built(name string) string {
	return fmt.Sprintf("%s %s (build %s, commit %s)", name, version, build, commit)
}
