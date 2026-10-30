//go:build linux

package process

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

var errCgroupUnavailable = errors.New("cgroup v2 unavailable")

func ConfigureSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid:   true,
		Pdeathsig: syscall.SIGTERM,
	}
}

func ApplyLimits(pid int, limits Limits) error {
	if limits.CPULimitSeconds > 0 {
		if err := setRlimit(pid, unix.RLIMIT_CPU, limits.CPULimitSeconds); err != nil {
			return fmt.Errorf("set cpu limit: %w", err)
		}
	}
	if limits.MemoryLimitBytes > 0 {
		if err := applyCgroupMemoryLimit(pid, limits.MemoryLimitBytes); err != nil && !errors.Is(err, errCgroupUnavailable) {
			return fmt.Errorf("set memory limit: %w", err)
		}
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
		return fmt.Errorf("%w: %w", errCgroupUnavailable, err)
	}
	pluginDir := filepath.Join(baseDir, fmt.Sprintf("numen-plugin-%d", pid))
	if err := os.Mkdir(pluginDir, 0o755); err != nil {
		return fmt.Errorf("%w: %w", errCgroupUnavailable, err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "cgroup.procs"), []byte(strconv.Itoa(pid)), 0o600); err != nil {
		return fmt.Errorf("%w: %w", errCgroupUnavailable, errors.Join(err, os.Remove(pluginDir)))
	}
	limitStr := strconv.FormatUint(limitBytes, 10)
	if err := os.WriteFile(filepath.Join(pluginDir, "memory.high"), []byte(limitStr), 0o600); err != nil {
		originalUnavailable := errors.Is(err, os.ErrPermission) || errors.Is(err, os.ErrNotExist)
		rollbackErr := os.WriteFile(filepath.Join(baseDir, "cgroup.procs"), []byte(strconv.Itoa(pid)), 0o600)
		removeErr := os.Remove(pluginDir)
		reportedErr := errors.Join(err, rollbackErr, removeErr)
		if originalUnavailable {
			return fmt.Errorf("%w: %w", errCgroupUnavailable, reportedErr)
		}
		return fmt.Errorf("write memory.high: %w", reportedErr)
	}
	return nil
}

func readCgroupMemory(pid int) (uint64, error) {
	baseDir, err := findCgroupV2Dir(pid)
	if err != nil {
		return 0, err
	}
	expectedName := fmt.Sprintf("numen-plugin-%d", pid)
	if filepath.Base(baseDir) != expectedName {
		return 0, errors.New("cgroup v2 directory not matched")
	}
	if bytes, err := readCgroupAnon(baseDir); err == nil {
		return bytes, nil
	}
	data, err := os.ReadFile(filepath.Join(baseDir, "memory.current"))
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
}

func readCgroupAnon(cgroupDir string) (uint64, error) {
	data, err := os.ReadFile(filepath.Join(cgroupDir, "memory.stat"))
	if err != nil {
		return 0, err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "anon ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				return strconv.ParseUint(fields[1], 10, 64)
			}
		}
	}
	return 0, errors.New("anon memory stat not found")
}

// terminateCgroupProcs kills every process of the plugin cgroup. A cgroup or process that is already gone is not an error.
func terminateCgroupProcs(pluginDir string) error {
	var failed []error
	if err := os.WriteFile(filepath.Join(pluginDir, "cgroup.kill"), []byte("1"), 0o600); err != nil && !errors.Is(err, os.ErrNotExist) {
		failed = append(failed, fmt.Errorf("write cgroup.kill: %w", err))
	}
	data, err := os.ReadFile(filepath.Join(pluginDir, "cgroup.procs"))
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			failed = append(failed, fmt.Errorf("read cgroup.procs: %w", err))
		}
		return errors.Join(failed...)
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if childPid, err := strconv.Atoi(line); err == nil && childPid > 0 {
			if err := syscall.Kill(childPid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
				failed = append(failed, fmt.Errorf("kill process %d: %w", childPid, err))
			}
		}
	}
	return errors.Join(failed...)
}

func CleanupCgroup(pid int) error {
	baseDir, err := findCgroupV2Dir(os.Getpid())
	if err != nil {
		return err
	}
	pluginDir := filepath.Join(baseDir, fmt.Sprintf("numen-plugin-%d", pid))
	terminateErr := terminateCgroupProcs(pluginDir)
	var removeErr error
	for attempt := 0; attempt < 5; attempt++ {
		removeErr = os.Remove(pluginDir)
		if removeErr == nil || errors.Is(removeErr, os.ErrNotExist) {
			return nil
		}
		terminateErr = terminateCgroupProcs(pluginDir)
	}
	return fmt.Errorf("remove plugin cgroup %s: %w", pluginDir, errors.Join(removeErr, terminateErr))
}

func ReadRSS(pid int) (uint64, error) {
	if bytes, err := readCgroupMemory(pid); err == nil && bytes > 0 {
		return bytes, nil
	}
	return readStatmRSS(pid)
}

func readStatmRSS(pid int) (uint64, error) {
	if bytes, err := readProcStatusAnon(pid); err == nil {
		return bytes, nil
	}
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/statm", pid))
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(data))
	if len(fields) >= 6 {
		pages, err := strconv.ParseUint(fields[5], 10, 64)
		if err == nil {
			return pages * uint64(os.Getpagesize()), nil
		}
	}
	if len(fields) < 2 {
		return 0, errors.New("invalid statm format")
	}
	pages, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse statm resident pages: %w", err)
	}
	return pages * uint64(os.Getpagesize()), nil
}

func readProcStatusAnon(pid int) (uint64, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0, err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "RssAnon:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, err := strconv.ParseUint(fields[1], 10, 64)
				if err != nil {
					return 0, fmt.Errorf("parse RssAnon: %w", err)
				}
				return kb * 1024, nil
			}
		}
	}
	return 0, errors.New("RssAnon not found")
}
