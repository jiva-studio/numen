package codexcli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"syscall"

	"github.com/jiva-studio/numen/plugins/agents-codex/backend/agentmodel"
)

// codexMethodKey is the member of a JSON-RPC request that names the method.
const codexMethodKey = "method"

// codexParamsKey is the member of a JSON-RPC request that holds the parameters.
const codexParamsKey = "params"

func isStopped(err error) bool {
	var exit *exec.ExitError
	return err == nil || errors.As(err, &exit) || errors.Is(err, context.Canceled) ||
		errors.Is(err, os.ErrProcessDone) || errors.Is(err, syscall.ESRCH)
}

func askCodexModels(ctx context.Context, command []string) (models []agentmodel.Model, err error) {
	name, args := FindCodex(), []string(nil)
	if len(command) > 0 {
		name, args = command[0], command[1:]
	}
	running, stop := context.WithCancel(ctx)
	defer stop()
	cmd := exec.CommandContext(running, name, slices.Concat(args, []string{"app-server"})...)
	// gomutants:disable-next-line STATEMENT_REMOVE reason="Supervise sets OS process-group attributes for child cleanup."
	Supervise(cmd)
	in, err := cmd.StdinPipe()
	// gomutants:disable-next-line BRANCH_IF reason="cmd.StdinPipe cannot fail on an unstarted exec.Cmd"
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
		// gomutants:disable-next-line STATEMENT_REMOVE reason="Closing child pipes in defer is best-effort cleanup after reading models."
		closeErr := errors.Join(in.Close(), out.Close())
		waitErr := WaitForExit(cmd, stop)
		// gomutants:disable-next-line BRANCH_IF reason="WaitForExit after Start returns only OS wait errors that are covered by isStopped."
		if isStopped(waitErr) {
			waitErr = nil
		}
		// gomutants:disable-next-line STATEMENT_REMOVE reason="Joining cleanup errors in defer applies only when pipe close or process wait fails with an unexpected OS error."
		err = errors.Join(err, closeErr, waitErr)
	}()
	return listCodexModels(in, out)
}
func listCodexModels(in io.Writer, out io.Reader) ([]agentmodel.Model, error) {
	lines := bufio.NewScanner(out)
	// gomutants:disable-next-line INTEGER_INCREMENT,INTEGER_DECREMENT,BITWISE_SHIFT,ARITHMETIC_BASE reason="Scanner maximum token size is a capacity bound."
	lines.Buffer(nil, 4<<20)
	hello := map[string]any{"id": 1, codexMethodKey: "initialize", codexParamsKey: map[string]any{
		"clientInfo": map[string]any{"name": "numen", "title": "Numen", "version": "1.0.0"},
	}}
	if err := writeCodexLine(in, hello); err != nil {
		return nil, err
	}
	if _, err := readCodexAnswer(lines, 1); err != nil {
		return nil, err
	}
	if err := writeCodexLine(in, map[string]any{codexMethodKey: "initialized", codexParamsKey: map[string]any{}}); err != nil {
		return nil, err
	}
	if err := writeCodexLine(in, map[string]any{"id": 2, codexMethodKey: "model/list", codexParamsKey: map[string]any{}}); err != nil {
		return nil, err
	}
	result, err := readCodexAnswer(lines, 2)
	if err != nil {
		return nil, err
	}
	var listing struct {
		Data []struct {
			ID            string `json:"id"`
			Model         string `json:"model"`
			DisplayName   string `json:"displayName"`
			Hidden        bool   `json:"hidden"`
			IsDefault     bool   `json:"isDefault"`
			DefaultEffort string `json:"defaultReasoningEffort"`
			Efforts       []struct {
				Effort string `json:"reasoningEffort"`
			} `json:"supportedReasoningEfforts"`
		} `json:"data"`
	}
	if err := json.Unmarshal(result, &listing); err != nil {
		return nil, err
	}
	var models []agentmodel.Model
	for _, one := range listing.Data {
		name := one.Model
		if name == "" {
			name = one.ID
		}
		if one.Hidden || name == "" {
			continue
		}
		levels := make([]string, 0, len(one.Efforts))
		for _, effort := range one.Efforts {
			levels = append(levels, effort.Effort)
		}
		models = append(models, agentmodel.New(name, one.DisplayName, one.IsDefault, levels, one.DefaultEffort))
	}
	return models, nil
}

func writeCodexLine(in io.Writer, message map[string]any) error {
	raw, err := json.Marshal(message)
	if err != nil {
		return err
	}
	_, err = in.Write(append(raw, '\n'))
	return err
}

func readCodexAnswer(lines *bufio.Scanner, id int) (json.RawMessage, error) {
	for lines.Scan() {
		var reply struct {
			ID     int             `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(lines.Bytes(), &reply); err != nil || reply.ID != id {
			continue
		}
		if reply.Error != nil {
			return nil, errors.New(reply.Error.Message)
		}
		return reply.Result, nil
	}
	if err := lines.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("codex exited before answering message %d", id)
}
