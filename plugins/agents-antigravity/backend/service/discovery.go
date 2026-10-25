package service

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	numenv1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
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
		return isRunnable(customPath[0])
	}
	found := FindAntigravity()
	return isRunnable(found)
}

// FindAntigravity returns the path of the Antigravity CLI, or the bare name where not found.
func FindAntigravity(customPath ...string) string {
	if len(customPath) > 0 && customPath[0] != "" {
		if isRunnable(customPath[0]) {
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
		if isRunnable(resolved) {
			return resolved
		}
	}
	return "agy"
}

func programCommand(commands []*numenv1.AgentProgramCommand, name string) []string {
	for _, one := range commands {
		if one.GetProgram() == name {
			return one.GetCommand()
		}
	}
	return nil
}

func isRunnable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		ext := strings.ToLower(filepath.Ext(path))
		return slices.Contains([]string{".exe", ".cmd", ".bat"}, ext)
	}
	return info.Mode()&0o111 != 0
}
