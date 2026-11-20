package process

import "golang.org/x/sys/unix"

// SetNice sets the plugin process to the profile's niceness.
func SetNice(pid, nice int) error { return unix.Setpriority(unix.PRIO_PROCESS, pid, nice) }
