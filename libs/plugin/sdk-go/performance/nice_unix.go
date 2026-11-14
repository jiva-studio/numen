//go:build linux || darwin

package performance

import (
	"cmp"
	"fmt"
	"os"
	"strconv"
	"syscall"
)

// SetNice sets the niceness of the process. On Linux the niceness belongs to each thread, so it
// sets every thread that exists; threads the process starts later inherit the niceness of the
// thread that starts them. An unprivileged process can only raise its niceness: a lower value than
// the one in force returns the error of the operating system, and the plugin keeps the niceness it
// has until it restarts.
const procTaskDir = "/proc/self/task"

func SetNice(nice int) error {
	return setNiceIn(procTaskDir, nice)
}

func setNiceIn(dir string, nice int) error {
	tasks, err := os.ReadDir(dir)
	if err != nil {
		return setPriority(0, nice)
	}
	var first error
	for _, task := range tasks {
		tid, err := strconv.Atoi(task.Name())
		if err != nil {
			continue
		}
		first = cmp.Or(first, setPriority(tid, nice))
	}
	return first
}

func setPriority(id, nice int) error {
	if err := syscall.Setpriority(syscall.PRIO_PROCESS, id, nice); err != nil {
		return fmt.Errorf("set niceness %d: %w", nice, err)
	}
	return nil
}
