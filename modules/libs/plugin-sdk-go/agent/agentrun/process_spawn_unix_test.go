//go:build unix

package agentrun_test

// spawning prints a line, then starts a child that outlives it unless the command is supervised.
var spawning = []string{"sh", "-c", "sleep 120 & echo started; wait"}

// isReadErrorFatal says whether the read of the output must end without an error.
const isReadErrorFatal = true
