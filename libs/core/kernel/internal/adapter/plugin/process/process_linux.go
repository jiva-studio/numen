//go:build linux

package process

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"sync"
	"syscall"

	"golang.org/x/sys/unix"
)

func ConfigureSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid:   true,
		Pdeathsig: syscall.SIGTERM,
	}
}

type startRequest struct {
	cmd    *exec.Cmd
	result chan<- error
}

var getStartRequests = sync.OnceValue(func() chan<- startRequest {
	requests := make(chan startRequest)
	go serveStartRequests(requests)
	return requests
})

func serveStartRequests(requests <-chan startRequest) {
	// Linux sends the parent-death signal when the thread that started the child ends. This goroutine
	// keeps its thread locked and never returns, so the thread ends only with the host.
	runtime.LockOSThread()
	for request := range requests {
		request.result <- request.cmd.Start()
	}
}

// StartCommand starts cmd from the one OS thread that starts every plugin process and lives as long as
// the host, and returns the error of cmd.Start.
func StartCommand(cmd *exec.Cmd) error {
	result := make(chan error, 1)
	getStartRequests() <- startRequest{cmd: cmd, result: result}
	return <-result
}

func ApplyLimits(pid int, limits Limits) error {
	if limits.CPULimitSeconds > 0 {
		if err := setRlimit(pid, unix.RLIMIT_CPU, limits.CPULimitSeconds); err != nil {
			return fmt.Errorf("set cpu limit: %w", err)
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

// HideInheritedFiles marks every open file descriptor above stderr close-on-exec,
// so a process started next inherits none of them. The window's asset server
// creates the pipe of each response without that flag: a plugin started while a
// response is open would keep the pipe's write end, and the page would not see
// the end of that response until the plugin exits.
func HideInheritedFiles() error {
	err := unix.CloseRange(3, ^uint(0), unix.CLOSE_RANGE_CLOEXEC)
	if errors.Is(err, unix.ENOSYS) {
		// An older kernel lacks close_range; those descriptors stay inherited.
		return nil
	}
	if err != nil {
		return fmt.Errorf("close_range: %w", err)
	}
	return nil
}
