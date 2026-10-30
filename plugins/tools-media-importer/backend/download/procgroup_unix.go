//go:build unix

package download

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// setProcessGroup puts the run in a process group of its own and makes cancelling it
// ask the whole group to stop, then force the group to stop after stopGrace.
func setProcessGroup(running *exec.Cmd) {
	running.SysProcAttr = new(syscall.SysProcAttr)
	running.SysProcAttr.Setpgid = true
	running.Cancel = func() error {
		group := -running.Process.Pid
		if err := syscall.Kill(group, syscall.SIGTERM); err != nil {
			if errors.Is(err, syscall.ESRCH) {
				return os.ErrProcessDone
			}
			return err
		}
		go func() {
			time.Sleep(stopGrace)
			if err := syscall.Kill(group, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
				fmt.Fprintf(os.Stderr, "tools-media-importer: stop the process group %d: %v\n", -group, err)
			}
		}()
		return nil
	}
	running.WaitDelay = stopGrace
}
