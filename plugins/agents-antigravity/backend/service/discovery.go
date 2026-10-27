package service

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
)

var antigravityPlaces = []string{
	"~/.local/bin/antigravity",
	"~/.local/bin/agy",
	"~/.antigravity/bin/antigravity",
	"/opt/homebrew/bin/antigravity",
	"/usr/local/bin/antigravity",
	"~/.nix-profile/bin/antigravity",
	"/run/current-system/sw/bin/antigravity",
}

// IsAntigravityInstalled checks whether Google Antigravity CLI is available on this machine.
func IsAntigravityInstalled(customPath ...string) bool {
	if len(customPath) > 0 && customPath[0] != "" {
		return agentrun.IsRunnable(customPath[0])
	}
	found := FindAntigravity()
	return agentrun.IsRunnable(found)
}

// FindAntigravity returns the path of the Antigravity CLI, or the bare name where not found.
func FindAntigravity(customPath ...string) string {
	if len(customPath) > 0 && customPath[0] != "" {
		if agentrun.IsRunnable(customPath[0]) {
			return customPath[0]
		}
	}
	for _, name := range []string{"agy", "antigravity"} {
		if found, err := exec.LookPath(name); err == nil {
			return found
		}
	}
	home, _ := os.UserHomeDir()
	for _, p := range antigravityPlaces {
		resolved := p
		if strings.HasPrefix(p, "~/") && home != "" {
			resolved = filepath.Join(home, p[2:])
		}
		if agentrun.IsRunnable(resolved) {
			return resolved
		}
	}
	return "agy"
}

func programCommand(commands []*agentrun.AgentProgramCommand, name string) []string {
	for _, one := range commands {
		if one.GetProgram() == name {
			return one.GetCommand()
		}
	}
	return nil
}
