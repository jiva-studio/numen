//go:build !(darwin || freebsd || linux || netbsd)

package recognition

// support is nothing where the runtime carries what it needs beside it.
func support() {}
