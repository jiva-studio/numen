package service

import "github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun"

var codexSearchPlaces = []string{
	"~/.local/bin/codex",
	"~/.codex/bin/codex",
	"~/.npm-global/bin/codex",
	"~/.bun/bin/codex",
	"/opt/homebrew/bin/codex",
	"/usr/local/bin/codex",
	"~/.nix-profile/bin/codex",
	"/run/current-system/sw/bin/codex",
}

// IsCodexInstalled checks whether OpenAI Codex CLI is available on this machine.
func IsCodexInstalled() bool { return agentrun.Locate([]string{"codex"}, codexSearchPlaces) != "" }

// FindCodex returns the path of the Codex CLI, or the bare name where not found.
func FindCodex() string {
	if found := agentrun.Locate([]string{"codex"}, codexSearchPlaces); found != "" {
		return found
	}
	return "codex"
}
