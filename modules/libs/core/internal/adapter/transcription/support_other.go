//go:build !(darwin || freebsd || linux || netbsd)

package transcription

// support is nothing where the runtime carries what it needs beside it.
func support() {}
