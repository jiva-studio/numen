package service

import "github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/discovery"

// programName is the name tasks use for the program, and the name of its executable.
const programName = "codex"

// codexProgram is where the Codex CLI is searched for.
var codexProgram = discovery.Program{
	Names: []string{programName},
	Places: []string{
		"~/.local/bin/codex",
		"~/.codex/bin/codex",
		"~/.npm-global/bin/codex",
		"~/.bun/bin/codex",
		"/opt/homebrew/bin/codex",
		"/usr/local/bin/codex",
		"~/.nix-profile/bin/codex",
		"/run/current-system/sw/bin/codex",
	},
	Fallback: programName,
}

// IsCodexInstalled checks whether OpenAI Codex CLI is available on this machine.
func IsCodexInstalled(customPath ...string) bool {
	return codexProgram.IsInstalled(customPath...)
}

// FindCodex returns the path of the Codex CLI, or the bare name where not found.
func FindCodex(customPath ...string) string {
	return codexProgram.Find(customPath...)
}
