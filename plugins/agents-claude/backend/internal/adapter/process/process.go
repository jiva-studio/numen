// Package process starts and ends the agent programs the plugin runs, each in a process group of
// its own.
package process

import (
	"os/exec"
	"time"
)

// waitDelay is how long the pipes of a killed child are waited for.
const waitDelay = 2 * time.Second

// ExitGrace is how long a program whose input has ended is given to exit before it is killed.
const ExitGrace = time.Second

// Supervise puts the command in a process group of its own, and ends the whole group when the
// command's context is canceled.
func Supervise(cmd *exec.Cmd) {
	detach(cmd)
	cmd.Cancel = func() error { return kill(cmd) }
	cmd.WaitDelay = waitDelay
}

// WaitForExit waits for the started command to exit, and calls stop, which cancels the command's
// context, if it is still running after ExitGrace.
//
// A program that exits on the end of its input is reaped by Wait before its group is signaled, so it
// finishes the way it chose to and is not reported as killed.
func WaitForExit(cmd *exec.Cmd, stop func()) error {
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	timer := time.NewTimer(ExitGrace)
	defer timer.Stop()
	select {
	case err := <-exited:
		return err
	case <-timer.C:
		stop()
		return <-exited
	}
}
