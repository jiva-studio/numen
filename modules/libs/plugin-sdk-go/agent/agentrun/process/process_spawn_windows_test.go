//go:build windows

package process_test

// spawning prints a line, then starts a child that outlives it unless the command is supervised.
var spawning = []string{"cmd", "/c", "echo started & ping -n 120 127.0.0.1"}

// isReadErrorFatal says whether the read of the output must end without an error. On windows
// the read may fail once the child is ended.
const isReadErrorFatal = false
