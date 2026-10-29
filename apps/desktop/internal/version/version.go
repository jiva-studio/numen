// Package version is what a build of this application calls itself.
//
// The two windows ship together, one version, one installer, and the linker
// writes the numbers into this package for both of them.
package version

import "fmt"

// The tag this build was made from, the number of commits since it, and the
// commit. A build made by hand carries these defaults.
var (
	version = "0.0.0-dev"
	build   = "0"
	commit  = "unknown"
)

// GetVersionLine returns the line the binary prints when queried: the version,
// the build number that distinguishes two builds of one version, and the commit
// the source was built from.
func GetVersionLine(name string) string {
	return fmt.Sprintf("%s %s (build %s, commit %s)", name, version, build, commit)
}
