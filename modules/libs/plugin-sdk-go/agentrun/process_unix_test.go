//go:build unix

package agentrun_test

import "syscall"

// isRunning reports whether a process this program did not wait for is still alive, without
// disturbing it. Signal 0 does nothing but ask.
func isRunning(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}
