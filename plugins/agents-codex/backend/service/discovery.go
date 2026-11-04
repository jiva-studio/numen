package service

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
)

// getCodexPlaces returns where the CLI is looked for after PATH. A leading ~/ is the home
// directory.
func getCodexPlaces() []string {
	return []string{
		"~/.local/bin/codex",
		"~/.codex/bin/codex",
		"~/.npm-global/bin/codex",
		"~/.bun/bin/codex",
		"/opt/homebrew/bin/codex",
		"/usr/local/bin/codex",
		"~/.nix-profile/bin/codex",
		"/run/current-system/sw/bin/codex",
	}
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
	return findCodexIn(getCodexPlaces(), customPath...)
}

// findCodexIn returns the path of the Codex CLI, searched in customPath, PATH and places, or the
// bare name where not found.
func findCodexIn(places []string, customPath ...string) string {
	if len(customPath) > 0 && agentrun.IsRunnable(customPath[0]) {
		return customPath[0]
	}
	if found, err := exec.LookPath("codex"); err == nil {
		return found
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	for _, p := range places {
		resolved := p
		if rest, ok := strings.CutPrefix(p, "~/"); ok && home != "" {
			resolved = filepath.Join(home, rest)
		}
		if agentrun.IsRunnable(resolved) {
			return resolved
		}
	}
	return "codex"
}

func programCommand(commands []agentrun.ProgramCommand, name string) []string {
	for _, one := range commands {
		if one.Program == name {
			return one.Command
		}
	}
	return nil
}
