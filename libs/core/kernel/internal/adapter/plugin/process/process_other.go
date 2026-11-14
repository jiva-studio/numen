//go:build !linux

package process

import "os/exec"

func ConfigureSysProcAttr(_ *exec.Cmd) {}

// Starter needs no dedicated thread here: only Linux ties the parent-death signal to the thread that started the child.
type Starter struct{}

func NewStarter() *Starter {
	return &Starter{}
}

func (*Starter) Start(cmd *exec.Cmd) error {
	return cmd.Start()
}

func ApplyLimits(_ int, _ Limits) error {
	return nil
}

// HideInheritedFiles does nothing where the window creates no pipes per response.
func HideInheritedFiles() error {
	return nil
}
