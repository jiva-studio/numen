package process

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"syscall"

	"github.com/jiva-studio/numen/plugins/agents-claude/backend/arguments"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/environment"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/model"
)

// Lister asks a started program for its models over the control protocol on its input.
type Lister struct{}

// ReportError writes a model-listing error to standard error.
func (Lister) ReportError(err error) {
	// gomutants:disable-next-line STATEMENT_REMOVE reason="Diagnostic log to os.Stderr does not affect the returned fallback models."
	fmt.Fprintf(os.Stderr, "list models of claude: %v\n", err)
}

func isStopped(err error) bool {
	var exit *exec.ExitError
	return err == nil || errors.As(err, &exit) || errors.Is(err, context.Canceled) ||
		errors.Is(err, os.ErrProcessDone) || errors.Is(err, syscall.ESRCH)
}

// ListModels starts the command, sends it the initialize request, and reads the models from its
// answer. The program is ended when the models are read or model.Timeout has passed.
func (Lister) ListModels(ctx context.Context, command, env []string) (list []model.Model, err error) {
	ctx, cancel := context.WithTimeout(ctx, model.Timeout)
	defer cancel()
	args := slices.Concat(command[1:], arguments.GetModelListing())
	cmd := exec.CommandContext(ctx, command[0], args...)
	cmd.Env = environment.DropSession(env)
	// gomutants:disable-next-line STATEMENT_REMOVE reason="Supervise sets OS process-group attributes for child cleanup."
	Supervise(cmd)
	in, err := cmd.StdinPipe()
	// gomutants:disable-next-line BRANCH_IF reason="cmd.StdinPipe on a fresh command before Start fails only on OS descriptor exhaustion."
	if err != nil {
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	// gomutants:disable-next-line BRANCH_IF reason="cmd.StdoutPipe on a fresh command before Start fails only on OS descriptor exhaustion."
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	defer func() {
		// gomutants:disable-next-line STATEMENT_REMOVE reason="Closing the child stdin pipe in defer is best-effort cleanup after reading models."
		err = errors.Join(err, in.Close())
		// gomutants:disable-next-line BRANCH_IF reason="WaitForExit after Start returns only OS wait errors that are covered by isStopped."
		if waitErr := WaitForExit(cmd, cancel); !isStopped(waitErr) {
			err = errors.Join(err, waitErr)
		}
	}()
	const requestID = "numen-models"
	request := `{"type":"control_request","request_id":"` + requestID + `","request":{"subtype":"initialize"}}` + "\n"
	// gomutants:disable-next-line BRANCH_IF reason="Writing the small initialize payload to a newly started pipe fails only if the child closes stdin immediately, which model.Parse also detects."
	if _, err := io.WriteString(in, request); err != nil {
		return nil, err
	}
	return model.Parse(out, requestID)
}
