//go:build !linux

package plugin

import "os/exec"

func configureSysProcAttr(_ *exec.Cmd) {}

func applyProcessLimits(_ int, _ ProcessLimits) error {
	return nil
}

func cleanupProcessCgroup(_ int) error {
	return nil
}

func readProcessRSS(_ int) (uint64, error) {
	return 0, nil
}
