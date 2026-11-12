// Package process runs external programs in process groups of their own.
package process

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
)

const stopGrace = 5 * time.Second

// Runner finds and runs external programs on the machine.
type Runner struct{}

// LookPath searches for an executable named file in the directories named by the PATH environment variable.
func (Runner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

// Environ returns a copy of strings representing the environment, in the form "key=value".
func (Runner) Environ() []string {
	return hostcall.GetChildEnvironment()
}

// Run executes command with arguments and env, streaming stdout to into when non-nil.
func (Runner) Run(
	ctx context.Context, command, env []string, into io.Writer, arguments ...string,
) (stdout []byte, combined string, err error) {
	running := exec.CommandContext(ctx, command[0],
		append(append([]string(nil), command[1:]...), arguments...)...)
	running.Env = env
	setProcessGroup(running)
	var out, response bytes.Buffer
	running.Stdout, running.Stderr = &out, &response
	if into != nil {
		running.Stdout = into
	}
	err = running.Run()
	if response.Len() > 0 {
		return out.Bytes(), response.String(), err
	}
	return out.Bytes(), out.String(), err
}

// ReportError writes a diagnostic message to standard error.
func (Runner) ReportError(what string, err error) {
	// gomutants:disable-next-line STATEMENT_REMOVE reason="Diagnostic log to os.Stderr has no observable effect on callers."
	fmt.Fprintf(os.Stderr, "tools-media-importer: %s: %v\n", what, err)
}
