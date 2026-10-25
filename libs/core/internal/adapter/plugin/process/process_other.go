//go:build !linux

package process

import "os/exec"

func ConfigureSysProcAttr(_ *exec.Cmd) {}

func ApplyLimits(_ int, _ Limits) error {
	return nil
}

func CleanupCgroup(_ int) error {
	return nil
}

func ReadRSS(_ int) (uint64, error) {
	return 0, nil
}
