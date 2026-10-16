//go:build !linux

package plugin

import "os/exec"

func configureSysProcAttr(_ *exec.Cmd) {}
