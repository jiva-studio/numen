package service

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
)

func writeMcpConfig(url, token string) (string, error) {
	if url == "" || token == "" {
		return "", nil
	}
	type server struct {
		Type    string            `json:"type"`
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
	}
	config := struct {
		Servers map[string]server `json:"mcpServers"`
	}{
		Servers: map[string]server{
			"numen": {
				Type:    "http",
				URL:     url,
				Headers: map[string]string{"Authorization": "Bearer " + token},
			},
		},
	}
	data, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("marshal mcp config: %w", err)
	}

	return agentrun.WriteTempFile("numen-tools-*.json", data)
}

// claudeArguments is what a run of the program is started from.
type claudeArguments struct {
	instructions  string
	model         string
	reasoning     string
	configuration string
	session       string
	turns         int
	tools         []string
	mayReadHooks  bool
}

// defaultTurns is how many times the program may go round on one task where the task names no limit.
const defaultTurns = 100

// buildClaudeArguments returns what the program is started with. A run that names no tool is one
// answer: the program is given no tool, no MCP server and one turn.
func buildClaudeArguments(in claudeArguments) []string {
	if len(in.tools) == 0 {
		return buildAnswerArguments(in)
	}
	turns := in.turns
	if turns <= 0 {
		turns = defaultTurns
	}

	args := []string{
		"-p",
		flagOutputFormat, formatStreamJSON,
		flagVerbose,
		"--include-partial-messages",
		"--strict-mcp-config",
		"--permission-mode", "dontAsk",
		"--max-turns", fmt.Sprint(turns),
	}

	if in.configuration != "" {
		args = append(args, "--mcp-config", in.configuration)
	}

	args = append(args, "--tools", "WebSearch")

	var allowed []string
	allowed = append(allowed, "WebSearch")
	for _, t := range in.tools {
		name := t
		if !strings.HasPrefix(name, "mcp__numen__") {
			name = "mcp__numen__" + name
		}
		allowed = append(allowed, name)
	}
	slices.Sort(allowed)
	args = append(args, "--allowedTools", strings.Join(allowed, ","))

	// A vault's own hooks and settings are never read. The person's own are read only when the
	// setting allows it.
	sources := ""
	if in.mayReadHooks {
		sources = "user"
	}
	args = append(args, "--setting-sources", sources)

	if in.instructions != "" {
		args = append(args, "--append-system-prompt", in.instructions)
	}
	if in.model != "" {
		args = append(args, "--model", in.model)
	}
	if in.reasoning != "" {
		args = append(args, "--effort", in.reasoning)
	}
	if in.session != "" {
		args = append(args, "--resume", in.session)
	}
	return args
}

// buildAnswerArguments returns what the program is started with to give one answer: no tool, no
// MCP server, no setting of this machine's and one turn.
func buildAnswerArguments(in claudeArguments) []string {
	args := []string{
		"-p",
		flagOutputFormat, formatStreamJSON,
		flagVerbose,
		"--include-partial-messages",
		"--strict-mcp-config",
		"--permission-mode", "dontAsk",
		"--max-turns", "1",
		"--tools", "",
		"--setting-sources", "",
	}
	if in.instructions != "" {
		args = append(args, "--append-system-prompt", in.instructions)
	}
	if in.model != "" {
		args = append(args, "--model", in.model)
	}
	if in.reasoning != "" {
		args = append(args, "--effort", in.reasoning)
	}
	if in.session != "" {
		args = append(args, "--resume", in.session)
	}
	return args
}
