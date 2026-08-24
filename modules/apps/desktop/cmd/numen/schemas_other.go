//go:build !linux

package main

// settled reports whether this machine holds the settings a folder picker
// reads. Only the toolkit on Linux is built on them.
func settled() bool { return true }
