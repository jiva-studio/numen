package codexcli

import (
	"fmt"
	"os"

	"github.com/jiva-studio/numen/plugins/agents-codex/backend/domain/reasoning"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/usecase/task"
)

// defaultModel is the model name that leaves the choice of the model to the program.
const defaultModel = "default"

// Commands builds the command lines of a Codex run.
type Commands struct{}

var _ task.Commands = Commands{}

func getDisabledFeatures() []string {
	return []string{
		"shell_tool", "unified_exec", "view_image", "multi_agent", "apps", "browser_use",
		"computer_use", "plugins", "image_generation", "hooks",
	}
}

func resolveProgram(program string) string {
	if program != "" {
		return program
	}
	return FindCodex()
}

// MakeToolsCommand returns the command of a run that may call the tools of the server of the spec. The
// program runs without approvals or a sandbox, and reads the token of the server from NUMEN_TOKEN.
func (Commands) MakeToolsCommand(spec task.Spec) task.Command {
	args := []string{
		"exec",
		"--json",
		"--dangerously-bypass-approvals-and-sandbox",
	}
	if spec.ToolsURL != "" {
		mcpURL := fmt.Sprintf(`mcp_servers.numen.url=%q`, spec.ToolsURL)
		args = append(args, "-c", mcpURL, "-c", `mcp_servers.numen.bearer_token_env_var="NUMEN_TOKEN"`)
	}
	args = appendModelArguments(args, spec)
	args = append(args, spec.Prompt)
	return task.Command{Path: resolveProgram(spec.Program), Args: args, Dir: "", Env: nil, Token: "", Release: nil}
}

// MakeAnswerCommand returns the command of a run that only answers, in a new empty folder that the
// release of the command removes.
func (Commands) MakeAnswerCommand(spec task.Spec) (task.Command, error) {
	dir, err := os.MkdirTemp("", "numen-codex-*")
	if err != nil {
		return task.Command{}, fmt.Errorf("create the folder to run in: %w", err)
	}
	args := []string{
		"exec",
		"--json",
		"--ephemeral",
		"--skip-git-repo-check",
		"--ignore-user-config",
		"--sandbox", "read-only",
		"-c", `web_search="disabled"`,
	}
	for _, feature := range getDisabledFeatures() {
		args = append(args, "--disable", feature)
	}
	args = appendModelArguments(args, spec)
	args = append(args, "--", spec.Prompt)
	return task.Command{
		Path: resolveProgram(spec.Program), Args: args, Dir: dir, Env: nil, Token: "",
		Release: func() error { return os.RemoveAll(dir) },
	}, nil
}

func appendModelArguments(args []string, spec task.Spec) []string {
	if model := spec.Model; model != "" && model != defaultModel {
		args = append(args, "--model", model)
	}
	if level := spec.Reasoning; reasoning.IsKnown(level) {
		args = append(args, "-c", fmt.Sprintf("model_reasoning_effort=%q", level))
	}
	return args
}
