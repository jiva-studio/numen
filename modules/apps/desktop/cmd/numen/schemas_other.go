//go:build !linux

package main

// findSchemas puts the settings a picker reads on the search path. Only the
// toolkit on Linux is built on them.
func findSchemas() {}

// settled reports whether this machine holds the settings a folder picker
// reads.
func settled() bool { return true }
