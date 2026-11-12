// Package agents knows the connection types of the plugin: the command that starts the ACP server of
// each, and the options of the session a type needs.
package agents

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
)

// The connection types of the plugin.
const (
	TypeACP         = "acp"
	TypeClaudeCode  = "claude-code"
	TypeCodex       = "codex"
	TypeAntigravity = "antigravity"
)

// acpSubcommand is the argument that starts the ACP server of several agents.
const acpSubcommand = "acp"

// claudeWebSearch is the one built-in tool of Claude Code a task may use.
const claudeWebSearch = "WebSearch"

// claudeTaskTurns is how many times Claude Code may go round on one task.
const claudeTaskTurns = 100

// serverName is the name the session offers the tools under, which Claude Code puts before their names.
const serverName = "numen"

var (
	// ErrNoProgram is the reason a connection names no program where its type requires one.
	ErrNoProgram = errors.New("the connection names no program")
	// ErrNotFound is the reason the program of a connection is not on the machine.
	ErrNotFound = errors.New("the program of the connection is not on the machine")
)

// Agent is a program that serves the Agent Client Protocol over its standard streams.
type Agent struct {
	Type    string
	Command []string
}

func getKnownBinaries(connectionType string) []string {
	switch connectionType {
	case TypeClaudeCode:
		return []string{"claude-agent-acp", "claude-code-acp"}
	case TypeCodex:
		return []string{"codex-acp"}
	case TypeAntigravity:
		return []string{"agy"}
	}
	return nil
}

func getArguments(connectionType, executable string) []string {
	if connectionType == TypeAntigravity {
		return []string{acpSubcommand}
	}
	if connectionType != TypeACP {
		return nil
	}
	base := filepath.Base(executable)
	switch strings.TrimSuffix(base, filepath.Ext(base)) {
	case "cursor-agent", "goose", "kimi", "opencode", "agy":
		return []string{acpSubcommand}
	case "gemini", "qwen":
		return []string{"--acp"}
	}
	return nil
}

// Find returns the agent of a connection of the type with the program field. An empty program is the
// first known binary of the type that lookPath finds; the type acp requires a program. A nil lookPath
// finds nothing.
func Find(connectionType, program string, lookPath func(string) (string, error)) (Agent, error) {
	program = strings.TrimSpace(program)
	candidates := []string{program}
	if program == "" {
		candidates = getKnownBinaries(connectionType)
	}
	if len(candidates) == 0 {
		return Agent{Type: connectionType, Command: nil}, ErrNoProgram
	}
	if lookPath == nil {
		return Agent{Type: connectionType, Command: nil}, ErrNotFound
	}
	for _, candidate := range candidates {
		found, err := lookPath(candidate)
		if err != nil || found == "" {
			continue
		}
		return Agent{Type: connectionType, Command: append([]string{found}, getArguments(connectionType, candidate)...)}, nil
	}
	return Agent{Type: connectionType, Command: nil}, ErrNotFound
}

// GetKey returns what tells the programs of two agents apart.
func (a Agent) GetKey() string {
	return a.Type + "\x00" + strings.Join(a.Command, "\x00")
}

// GetTaskMeta returns the _meta of session/new for a task whose tools the session serves under served.
// Claude Code gets its own tools turned off but web search, the served tools allowed without asking, no
// settings, hooks or skills read, and a limit of turns; other types get none.
func (a Agent) GetTaskMeta(served []string) map[string]any {
	if a.Type != TypeClaudeCode {
		return nil
	}
	allowed := []string{claudeWebSearch}
	for _, name := range served {
		allowed = append(allowed, "mcp__"+serverName+"__"+name)
	}
	slices.Sort(allowed)
	return writeClaudeOptions(map[string]any{
		"tools":          []string{claudeWebSearch},
		"allowedTools":   allowed,
		"settingSources": []string{},
		"maxTurns":       claudeTaskTurns,
	})
}

// GetAnswerMeta returns the _meta of session/new for a session that answers with no tool: Claude Code
// gets every tool of its own turned off, no settings read and one turn; other types get none.
func (a Agent) GetAnswerMeta() map[string]any {
	if a.Type != TypeClaudeCode {
		return nil
	}
	return writeClaudeOptions(map[string]any{
		"tools":          []string{},
		"settingSources": []string{},
		"maxTurns":       1,
	})
}

func writeClaudeOptions(options map[string]any) map[string]any {
	return map[string]any{"claudeCode": map[string]any{"options": options}}
}
