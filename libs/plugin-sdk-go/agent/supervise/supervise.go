// Package supervise starts the child programs of an agent plugin in a process group of their own, so
// that ending the command ends every process the child started.
package supervise

import (
	"os/exec"
	"time"
)

// waitDelay is how long the pipes of a killed child are waited for.
const waitDelay = 2 * time.Second

// Command puts the command in a process group of its own, and ends the whole group when the
// command's context is cancelled.
func Command(cmd *exec.Cmd) {
	detach(cmd)
	cmd.Cancel = func() error { return kill(cmd) }
	cmd.WaitDelay = waitDelay
}
