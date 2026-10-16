//go:build linux

package plugin

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

func configureSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid:   true,
		Pdeathsig: syscall.SIGTERM,
	}
}

func applyProcessLimits(pid int, limits ProcessLimits) error {
	if limits.CPULimitSeconds > 0 {
		if err := setRlimit(pid, unix.RLIMIT_CPU, limits.CPULimitSeconds); err != nil {
			return fmt.Errorf("set cpu limit: %w", err)
		}
	}
	if limits.MemoryLimitBytes > 0 {
		_ = applyCgroupMemoryLimit(pid, limits.MemoryLimitBytes)
	}
	return nil
}

func setRlimit(pid int, resource int, limit uint64) error {
	var old unix.Rlimit
	if err := unix.Prlimit(pid, resource, nil, &old); err != nil {
		return err
	}
	newLim := old
	if limit < old.Max || old.Max == unix.RLIM_INFINITY {
		newLim.Cur = limit
		if limit < old.Max {
			newLim.Max = limit
		}
	} else {
		newLim.Cur = old.Max
	}
	return unix.Prlimit(pid, resource, &newLim, nil)
}

func findCgroupV2Dir(pid int) (string, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cgroup", pid))
	if err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "0::") {
			relPath := strings.TrimPrefix(line, "0::")
			return filepath.Join("/sys/fs/cgroup", relPath), nil
		}
	}
	return "", errors.New("cgroup v2 not found")
}

func applyCgroupMemoryLimit(pid int, limitBytes uint64) error {
	baseDir, err := findCgroupV2Dir(pid)
	if err != nil {
		return err
	}
	pluginDir := filepath.Join(baseDir, fmt.Sprintf("numen-plugin-%d", pid))
	if err := os.Mkdir(pluginDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "cgroup.procs"), []byte(strconv.Itoa(pid)), 0o600); err != nil {
		_ = os.Remove(pluginDir)
		return err
	}
	limitStr := strconv.FormatUint(limitBytes, 10)
	_ = os.WriteFile(filepath.Join(pluginDir, "memory.high"), []byte(limitStr), 0o600)
	_ = os.WriteFile(filepath.Join(pluginDir, "memory.max"), []byte(limitStr), 0o600)
	return nil
}

func readCgroupMemory(pid int) (uint64, error) {
	baseDir, err := findCgroupV2Dir(pid)
	if err != nil {
		return 0, err
	}
	pluginDir := filepath.Join(baseDir, fmt.Sprintf("numen-plugin-%d", pid))
	data, err := os.ReadFile(filepath.Join(pluginDir, "memory.current"))
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
}

func cleanupProcessCgroup(pid int) {
	baseDir, err := findCgroupV2Dir(os.Getpid())
	if err != nil {
		return
	}
	pluginDir := filepath.Join(baseDir, fmt.Sprintf("numen-plugin-%d", pid))
	_ = os.Remove(pluginDir)
}

func readProcessRSS(pid int) (uint64, error) {
	if bytes, err := readCgroupMemory(pid); err == nil && bytes > 0 {
		return bytes, nil
	}
	return readStatmRSS(pid)
}

func readStatmRSS(pid int) (uint64, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/statm", pid))
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return 0, errors.New("invalid statm format")
	}
	pages, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse statm resident pages: %w", err)
	}
	return pages * uint64(os.Getpagesize()), nil
}
