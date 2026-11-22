//go:build linux

package process

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"golang.org/x/sys/unix"
)

// SetNice updates the plugin threads. A second snapshot covers threads created during the first pass.
func SetNice(pid, nice int) error {
	dir := fmt.Sprintf("/proc/%d/task", pid)
	for range 2 {
		if err := setThreadPriorities(dir, nice); err != nil {
			return err
		}
	}
	return nil
}

func setThreadPriorities(dir string, nice int) error {
	threads, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, thread := range threads {
		tid, err := strconv.Atoi(thread.Name())
		if err != nil {
			return err
		}
		if err := unix.Setpriority(unix.PRIO_PROCESS, tid, nice); err != nil && !errors.Is(err, unix.ESRCH) {
			return markNiceDenied(err)
		}
	}
	return nil
}
