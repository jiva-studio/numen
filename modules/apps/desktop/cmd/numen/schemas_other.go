//go:build !linux

package main

// findSchemas puts the settings a dialog reads on the search path. Only the
// toolkit on Linux is built on them.
func findSchemas() {}

// settled reports whether this machine holds the settings a folder dialog
// reads.
func settled() bool { return true }
