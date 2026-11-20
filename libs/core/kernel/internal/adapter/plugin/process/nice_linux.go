//go:build linux

package process

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"golang.org/x/sys/unix"
)

// SetNice sets every thread of the plugin process to the profile's niceness.
// New threads inherit the priority of the thread that creates them.
func SetNice(pid, nice int) error {
	dir := fmt.Sprintf("/proc/%d/task", pid)
	applied := map[string]bool{}
	// The second snapshot covers threads created during the first pass.
	for range 2 {
		threads, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		changed := false
		for _, thread := range threads {
			if applied[thread.Name()] {
				continue
			}
			tid, err := strconv.Atoi(thread.Name())
			if err != nil {
				return err
			}
			if err := unix.Setpriority(unix.PRIO_PROCESS, tid, nice); err != nil && !errors.Is(err, unix.ESRCH) {
				return err
			}
			applied[thread.Name()] = true
			changed = true
		}
		if !changed {
			return nil
		}
	}
	return nil
}
