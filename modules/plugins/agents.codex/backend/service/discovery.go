package service

import "github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun"

var codexProgram = agentrun.Program{
	Names: []string{"codex"},
	Places: []string{
		"~/.codex/bin/codex",
		"~/.npm-global/bin/codex",
		"~/.bun/bin/codex",
	},
}

// IsCodexInstalled checks whether OpenAI Codex CLI is available on this machine.
func IsCodexInstalled() bool {
	_, ok := agentrun.Discover(codexProgram)
	return ok
}

// FindCodex returns the path of the Codex CLI, or the bare name where not found.
func FindCodex() string {
	path, _ := agentrun.Discover(codexProgram)
	return path
}
