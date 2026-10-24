//go:build windows

package service

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

func detach(*exec.Cmd) {}

func kill(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	ended := exec.Command(taskkill(), "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
	if ended == nil {
		return nil
	}
	if err := cmd.Process.Kill(); err != nil {
		return err
	}
	return ended
}

func taskkill() string {
	return filepath.Join(os.Getenv("SystemRoot"), "System32", "taskkill.exe")
}
