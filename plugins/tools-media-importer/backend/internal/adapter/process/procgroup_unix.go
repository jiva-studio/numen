//go:build unix

package process

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

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
			// gomutants:disable-next-line EXPRESSION_REMOVE,CONDITIONALS_NEGATION,INVERT_LOGICAL,REMOVE_LOGICAL_NOT reason="stderr log after 5s stopGrace in detached goroutine does not affect callers"
			if err := syscall.Kill(group, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
				fmt.Fprintf(os.Stderr, "tools-media-importer: stop the process group %d: %v\n", -group, err)
			}
		}()
		return nil
	}
	running.WaitDelay = stopGrace
}
