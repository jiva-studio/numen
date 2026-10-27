package service

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
)

var codexPlaces = []string{
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
func IsCodexInstalled(customPath ...string) bool {
	if len(customPath) > 0 && customPath[0] != "" {
		return agentrun.IsRunnable(customPath[0])
	}
	p := FindCodex()
	return agentrun.IsRunnable(p)
}

// FindCodex returns the path of the Codex CLI, or the bare name where not found.
func FindCodex(customPath ...string) string {
	if len(customPath) > 0 && customPath[0] != "" {
		if agentrun.IsRunnable(customPath[0]) {
			return customPath[0]
		}
	}
	if found, err := exec.LookPath("codex"); err == nil {
		return found
	}
	home, _ := os.UserHomeDir()
	for _, p := range codexPlaces {
		resolved := p
		if strings.HasPrefix(p, "~/") && home != "" {
			resolved = filepath.Join(home, p[2:])
		}
		if agentrun.IsRunnable(resolved) {
			return resolved
		}
	}
	return "codex"
}

func programCommand(commands []*agentrun.AgentProgramCommand, name string) []string {
	for _, one := range commands {
		if one.GetProgram() == name {
			return one.GetCommand()
		}
	}
	return nil
}
