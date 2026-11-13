package antigravity

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/model"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/usecase/task"
)

const (
	// flagPrint runs one turn without a prompt on the command line: the prompt is the message on the
	// standard input.
	flagPrint        = "--print="
	flagInputFormat  = "--input-format"
	flagOutputFormat = "--output-format"
	streamJSON       = "stream-json"
)

// inputMessage is a message of the stream-json input of the program.
type inputMessage struct {
	Event   string           `json:"event"`
	Message inputUserMessage `json:"message"`
}

type inputUserMessage struct {
	Content string `json:"content"`
}

func getPrintArgs() []string {
	return []string{flagPrint, flagInputFormat, streamJSON, flagOutputFormat, streamJSON}
}

func writeInput(prompt string) (string, error) {
	line, err := json.Marshal(inputMessage{Event: "user", Message: inputUserMessage{Content: prompt}})
	if err != nil {
		return "", fmt.Errorf("encode the prompt: %w", err)
	}
	return string(line) + "\n", nil
}

func makeTaskCommand(path string, run task.Run) (task.Command, error) {
	if run.ToolsURL == "" {
		return makeAnswerCommand(path, run)
	}
	args := appendRunFlags(append(getPrintArgs(), "--dangerously-skip-permissions"), run)
	return makeCommand(path, args, run.Prompt, &endpoint{URL: run.ToolsURL, Token: run.ToolsToken})
}

func makeAnswerCommand(path string, run task.Run) (task.Command, error) {
	return makeCommand(path, appendRunFlags(getPrintArgs(), run), run.Prompt, nil)
}

func makeCommand(path string, args []string, prompt string, tools *endpoint) (task.Command, error) {
	input, err := writeInput(prompt)
	if err != nil {
		return task.Command{}, err
	}
	tempHome, cleanHome, err := setupAntigravityHome(tools)
	if err != nil {
		return task.Command{}, fmt.Errorf("set up the Antigravity home: %w", err)
	}
	dir, cleanDir, err := createTempDir("numen-antigravity-run-*")
	if err != nil {
		return task.Command{}, errors.Join(fmt.Errorf("create the folder to run in: %w", err), cleanHome())
	}
	return task.Command{
		Path: path, Args: args, Dir: dir, Env: append(hostcall.GetChildEnvironment(), "HOME="+tempHome), Stdin: input,
		Release: func() error { return errors.Join(cleanHome(), cleanDir()) },
	}, nil
}

func appendRunFlags(args []string, run task.Run) []string {
	if run.Model != "" && run.Model != model.DefaultName {
		args = append(args, "--model", run.Model)
	}
	if run.Session != "" {
		args = append(args, "--conversation", run.Session)
	}
	return args
}
