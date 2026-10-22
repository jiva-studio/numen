// Package process supervises the child processes an agent program starts.
package process

import (
	"os/exec"
	"time"
)

// waitDelay is how long the pipes of a killed child are waited for. A grandchild holding the
// child's output keeps a wait from returning.
const waitDelay = 2 * time.Second

// Supervise puts the command in a process group of its own, and ends the whole group when the
// command's context is cancelled.
func Supervise(cmd *exec.Cmd) {
	detach(cmd)
	cmd.Cancel = func() error { return kill(cmd) }
	cmd.WaitDelay = waitDelay
}
