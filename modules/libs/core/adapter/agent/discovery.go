package agent

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var claudeSearchPlaces = []string{
	"~/.local/bin/claude",
	"~/.claude/local/claude",
	"~/Applications/Claude Code URL Handler.app/Contents/MacOS/claude",
	"/Applications/Claude Code URL Handler.app/Contents/MacOS/claude",
	"~/.bun/bin/claude",
	"~/.volta/bin/claude",
	"~/.npm-global/bin/claude",
	"~/.nvm/versions/node/*/bin/claude",
	"~/.nix-profile/bin/claude",
	"/opt/homebrew/bin/claude",
	"/usr/local/bin/claude",
	"/run/current-system/sw/bin/claude",
	"/nix/var/nix/profiles/default/bin/claude",
}

var antigravitySearchPlaces = []string{
	"~/.local/bin/agy",
	"~/.local/bin/antigravity",
	"~/.gemini/antigravity-cli/bin/agy",
	"~/.antigravity/bin/agy",
	"/opt/homebrew/bin/agy",
	"/opt/homebrew/bin/antigravity",
	"/usr/local/bin/agy",
	"/usr/local/bin/antigravity",
	"~/.nix-profile/bin/agy",
	"~/.nix-profile/bin/antigravity",
	"/run/current-system/sw/bin/agy",
	"/run/current-system/sw/bin/antigravity",
	"/etc/profiles/per-user/*/bin/antigravity",
	"/etc/profiles/per-user/*/bin/agy",
}

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

// IsClaudeInstalled checks whether Claude Code CLI is available on this machine.
func IsClaudeInstalled() bool {
	if _, err := exec.LookPath("claude"); err == nil {
		return true
	}
	return findInSearchPlaces(claudeSearchPlaces) != ""
}

// IsAntigravityInstalled checks whether Antigravity CLI is available on this machine.
func IsAntigravityInstalled() bool {
	if _, err := exec.LookPath("agy"); err == nil {
		return true
	}
	if _, err := exec.LookPath("antigravity"); err == nil {
		return true
	}
	return findInSearchPlaces(antigravitySearchPlaces) != ""
}

// IsCodexInstalled checks whether OpenAI Codex CLI is available on this machine.
func IsCodexInstalled() bool {
	if _, err := exec.LookPath("codex"); err == nil {
		return true
	}
	return findInSearchPlaces(codexSearchPlaces) != ""
}

func findInSearchPlaces(places []string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	for _, place := range places {
		if strings.HasPrefix(place, "~/") {
			if home == "" {
				continue
			}
			place = filepath.Join(home, place[2:])
		}
		matches, err := filepath.Glob(place)
		if err != nil {
			continue
		}
		for _, match := range matches {
			if isExecutable(match) {
				return match
			}
		}
	}
	return ""
}

func isExecutable(path string) bool {
	about, err := os.Stat(path)
	if err != nil || about.IsDir() {
		return false
	}
	return about.Mode().Perm()&0o111 != 0
}
