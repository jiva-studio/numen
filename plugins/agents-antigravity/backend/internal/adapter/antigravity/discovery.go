package antigravity

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
)

func getPlaces() []string {
	return []string{
		"~/.local/bin/antigravity",
		"~/.local/bin/agy",
		"~/.antigravity/bin/antigravity",
		"/opt/homebrew/bin/antigravity",
		"/usr/local/bin/antigravity",
		"~/.nix-profile/bin/antigravity",
		"/run/current-system/sw/bin/antigravity",
	}
}

func isInstalled(customPath ...string) bool {
	if len(customPath) > 0 && customPath[0] != "" {
		return agentrun.IsRunnable(customPath[0])
	}
	found := findProgram()
	return agentrun.IsRunnable(found)
}

func findProgram(customPath ...string) string {
	return findIn(getPlaces(), customPath...)
}

func findIn(places []string, customPath ...string) string {
	if len(customPath) > 0 && agentrun.IsRunnable(customPath[0]) {
		return customPath[0]
	}
	for _, name := range []string{"agy", "antigravity"} {
		if found, err := exec.LookPath(name); err == nil {
			return found
		}
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
	return "agy"
}
