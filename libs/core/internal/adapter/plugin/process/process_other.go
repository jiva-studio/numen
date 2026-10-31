//go:build !linux

package process

import "os/exec"

func ConfigureSysProcAttr(_ *exec.Cmd) {}

func ApplyLimits(_ int, _ Limits) error {
	return nil
}
