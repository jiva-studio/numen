//go:build unix

package process

import (
	"os/exec"
	"testing"
)

func TestSuperviseDetachesAndSetsCancelOnUnix(t *testing.T) {
	cmd := exec.CommandContext(t.Context(), "true")
	cmd.Cancel = nil
	Supervise(cmd)
	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.Setpgid {
		t.Fatalf("SysProcAttr = %+v, want Setpgid=true", cmd.SysProcAttr)
	}
	if cmd.Cancel == nil {
		t.Fatal("Supervise did not set cmd.Cancel")
	}
	if err := cmd.Cancel(); err != nil {
		t.Fatalf("cmd.Cancel() on unstarted process = %v, want nil", err)
	}
}
