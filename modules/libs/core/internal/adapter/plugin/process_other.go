//go:build !linux

package plugin

import "os/exec"

func configureSysProcAttr(_ *exec.Cmd) {}

func applyProcessLimits(_ int, _ ProcessLimits) error {
	return nil
}
