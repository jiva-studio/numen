//go:build unix

package claudecode_test

import "syscall"

// isRunning reports whether a process not started by this program is still
// alive, without disturbing it. Signal 0 does nothing but ask.
func isRunning(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}
