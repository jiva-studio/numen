//go:build !linux

package process

import "os/exec"

func ConfigureSysProcAttr(_ *exec.Cmd) {}

// StartCommand needs no dedicated thread here: only Linux ties the parent-death signal to the thread that started the child.
func StartCommand(cmd *exec.Cmd) error {
	return cmd.Start()
}

func ApplyLimits(_ int, _ Limits) error {
	return nil
}

// HideInheritedFiles does nothing where the window creates no pipes per response.
func HideInheritedFiles() error {
	return nil
}
