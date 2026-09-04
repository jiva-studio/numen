//go:build !unix

package claudecode_test

import "os"

// running reports whether a process not started by this program is still
// alive, without disturbing it. Opening it is itself the ask: there is
// nothing here that only asks.
func running(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	proc.Release()
	return true
}
