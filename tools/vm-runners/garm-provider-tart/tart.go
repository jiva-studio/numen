package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

const (
	sharedDirName       = "garm"
	stopDeadlineSeconds = 60
)

type commandRunner interface {
	runCommand(ctx context.Context, name string, args ...string) ([]byte, error)
	startDetached(ctx context.Context, name string, args []string, logPath string) (int, error)
}

type execRunner struct{}

func (execRunner) runCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), "TART_NO_AUTO_PRUNE=1")
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			return out, fmt.Errorf("%s %v: %w: %s", name, args, err, exitErr.Stderr)
		}
		return out, fmt.Errorf("%s %v: %w", name, args, err)
	}
	return out, nil
}

func (execRunner) startDetached(ctx context.Context, name string, args []string, logPath string) (int, error) {
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return 0, fmt.Errorf("open %s: %w", logPath, err)
	}
	cmd := exec.CommandContext(context.WithoutCancel(ctx), name, args...)
	cmd.Env = append(os.Environ(), "TART_NO_AUTO_PRUNE=1")
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	startErr := cmd.Start()
	closeErr := logFile.Close()
	if startErr != nil {
		return 0, fmt.Errorf("start %s %v: %w", name, args, startErr)
	}
	pid := cmd.Process.Pid
	if err := cmd.Process.Release(); err != nil {
		return pid, fmt.Errorf("release process %d: %w", pid, err)
	}
	if closeErr != nil {
		return pid, fmt.Errorf("close %s: %w", logPath, closeErr)
	}
	return pid, nil
}

type tartVM struct {
	Name    string `json:"Name"`
	State   string `json:"State"`
	Running bool   `json:"Running"`
}

type tartClient struct {
	runner commandRunner
	binary string
}

func (t tartClient) listVMs(ctx context.Context) (map[string]tartVM, error) {
	out, err := t.runner.runCommand(ctx, t.binary, "list", "--source", "local", "--format", "json")
	if err != nil {
		return nil, err
	}
	var vms []tartVM
	if err := json.Unmarshal(out, &vms); err != nil {
		return nil, fmt.Errorf("parse tart list: %w", err)
	}
	byName := make(map[string]tartVM, len(vms))
	for _, vm := range vms {
		byName[vm.Name] = vm
	}
	return byName, nil
}

func (t tartClient) cloneVM(ctx context.Context, source, name string) error {
	_, err := t.runner.runCommand(ctx, t.binary, "clone", source, name)
	return err
}

func (t tartClient) setResources(ctx context.Context, name string, cpu, memory int) error {
	args := []string{"set", name, "--random-mac"}
	if cpu > 0 {
		args = append(args, "--cpu", strconv.Itoa(cpu))
	}
	if memory > 0 {
		args = append(args, "--memory", strconv.Itoa(memory))
	}
	_, err := t.runner.runCommand(ctx, t.binary, args...)
	return err
}

func (t tartClient) startVM(ctx context.Context, name, sharedDir, logPath string) (int, error) {
	args := []string{"run", "--no-graphics", "--no-audio", "--dir=" + sharedDirName + ":" + sharedDir, name}
	return t.runner.startDetached(ctx, t.binary, args, logPath)
}

func (t tartClient) stopVM(ctx context.Context, name string) error {
	_, err := t.runner.runCommand(ctx, t.binary, "stop", name, "--timeout", strconv.Itoa(stopDeadlineSeconds))
	return err
}

func (t tartClient) deleteVM(ctx context.Context, name string) error {
	_, err := t.runner.runCommand(ctx, t.binary, "delete", name)
	return err
}
