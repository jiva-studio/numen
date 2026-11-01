package service

import (
	"context"
	"os/exec"
	"time"
)

// waitDelay is how long the pipes of a killed child are waited for.
const waitDelay = 2 * time.Second

// exitGrace is how long a program whose input has ended is given to exit before it is killed.
const exitGrace = time.Second

// supervise puts the command in a process group of its own, and ends the whole group when the
// command's context is cancelled.
func supervise(cmd *exec.Cmd) {
	detach(cmd)
	cmd.Cancel = func() error { return kill(cmd) }
	cmd.WaitDelay = waitDelay
}

// waitForExit waits for the started command to exit, and calls stop, which cancels the command's
// context, if it is still running after exitGrace.
//
// A program that exits on the end of its input is reaped by Wait before its group is signalled, so it
// finishes the way it chose to and is not reported as killed.
func waitForExit(cmd *exec.Cmd, stop func()) error {
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	grace, release := context.WithTimeout(context.Background(), exitGrace)
	defer release()
	select {
	case err := <-exited:
		return err
	case <-grace.Done():
		stop()
		return <-exited
	}
}
