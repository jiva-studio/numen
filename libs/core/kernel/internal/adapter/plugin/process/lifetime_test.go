//go:build unix

package process

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"
)

const (
	lifetimeRoleEnv       = "NUMEN_TEST_LIFETIME_ROLE"
	lifetimePluginRoleEnv = "NUMEN_TEST_LIFETIME_PLUGIN_ROLE"

	roleHost          = "host"
	roleReadingPlugin = "reading-plugin"
	roleIdlePlugin    = "idle-plugin"

	startedLine = "started"
)

// TestLifetimeHelper is the body of the host and plugin processes the lifetime tests start; it does
// nothing in a normal run. A plugin process holds file descriptor 3, the write end of a pipe the test
// reads: the read end reports end of input once the plugin has exited.
func TestLifetimeHelper(t *testing.T) {
	switch os.Getenv(lifetimeRoleEnv) {
	case roleHost:
		runLifetimeHost(t, os.Getenv(lifetimePluginRoleEnv))
	case roleReadingPlugin:
		if _, err := io.Copy(io.Discard, os.Stdin); err != nil {
			t.Fatal(err)
		}
	case roleIdlePlugin:
		time.Sleep(time.Minute)
	}
}

func runLifetimeHost(t *testing.T, pluginRole string) {
	cmd := newLifetimePlugin(t.Context(), pluginRole, os.NewFile(3, "plugin exit pipe"))
	input, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := StartCommand(cmd); err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(os.Stdout, startedLine+"\n"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Minute)
	runtime.KeepAlive(input)
}

func newLifetimePlugin(ctx context.Context, role string, exitPipe *os.File) *exec.Cmd {
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestLifetimeHelper$")
	cmd.Env = append(os.Environ(), lifetimeRoleEnv+"="+role)
	cmd.ExtraFiles = []*os.File{exitPipe}
	ConfigureSysProcAttr(cmd)
	return cmd
}

func isClosedWithin(t *testing.T, r *os.File, timeout time.Duration) bool {
	t.Helper()
	closed := make(chan error, 1)
	go func() {
		_, err := r.Read(make([]byte, 1))
		closed <- err
	}()
	select {
	case err := <-closed:
		if !errors.Is(err, io.EOF) {
			t.Fatalf("read the plugin exit pipe: %v", err)
		}
		return true
	case <-time.After(timeout):
		return false
	}
}

func TestKillingTheHostStopsThePlugin(t *testing.T) {
	for _, tc := range []struct {
		name       string
		pluginRole string
		linuxOnly  bool
	}{
		{"a plugin that reads its input until it ends", roleReadingPlugin, false},
		{"a plugin that ignores its input", roleIdlePlugin, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.linuxOnly && runtime.GOOS != "linux" {
				t.Skip("only Linux stops a child when its parent exits")
			}
			exitRead, exitWrite, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := exitRead.Close(); err != nil {
					t.Error(err)
				}
			}()

			host := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestLifetimeHelper$")
			host.Env = append(os.Environ(), lifetimeRoleEnv+"="+roleHost, lifetimePluginRoleEnv+"="+tc.pluginRole)
			host.ExtraFiles = []*os.File{exitWrite}
			output, err := host.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := host.Start(); err != nil {
				t.Fatal(err)
			}
			if err := exitWrite.Close(); err != nil {
				t.Fatal(err)
			}
			waitForStartedLine(t, output)

			if err := host.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			if err := host.Wait(); err == nil {
				t.Fatal("the killed host exited without an error")
			}
			if !isClosedWithin(t, exitRead, 10*time.Second) {
				t.Fatal("the plugin still runs 10 seconds after the host was killed")
			}
		})
	}
}

func waitForStartedLine(t *testing.T, output io.Reader) {
	t.Helper()
	lines := bufio.NewScanner(output)
	for lines.Scan() {
		if lines.Text() == startedLine {
			return
		}
	}
	t.Fatalf("the host exited before it started the plugin: %v", lines.Err())
}
