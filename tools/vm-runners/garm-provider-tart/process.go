package main

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type processTable struct {
	runner commandRunner
}

func (t processTable) getCommandLine(ctx context.Context, pid int) (string, bool) {
	out, err := t.runner.runCommand(ctx, "ps", "-ww", "-p", strconv.Itoa(pid), "-o", "command=")
	line := strings.TrimSpace(string(out))
	if err != nil || line == "" {
		return "", false
	}
	return line, true
}

func (t processTable) killHard(ctx context.Context, pid int) error {
	_, err := t.runner.runCommand(ctx, "kill", "-9", strconv.Itoa(pid))
	return err
}

func isMentioned(commandLine, marker string) bool {
	pattern := `(^|[^\w/.-])` + regexp.QuoteMeta(marker) + `($|[^\w-])`
	return regexp.MustCompile(pattern).MatchString(commandLine)
}

func readPID(pidPath string) (int, bool) {
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0, false
	}
	return pid, true
}

func (p provider) getRunningPID(ctx context.Context, state instanceState) (int, bool) {
	pid, ok := readPID(filepath.Join(p.store.getInstanceDir(state.Name), "tart-run.pid"))
	if !ok {
		return 0, false
	}
	line, ok := p.processes.getCommandLine(ctx, pid)
	if !ok || !isMentioned(line, state.VM) {
		return 0, false
	}
	return pid, true
}

func (p provider) stopVM(ctx context.Context, state instanceState) error {
	stopErr := p.tart.stopVM(ctx, state.VM)
	for range p.exitWaitSeconds {
		if _, running := p.getRunningPID(ctx, state); !running {
			return nil
		}
		time.Sleep(time.Second)
	}
	pid, running := p.getRunningPID(ctx, state)
	if !running {
		return nil
	}
	if err := p.processes.killHard(ctx, pid); err != nil {
		if stopErr != nil {
			return stopErr
		}
		return err
	}
	return nil
}
