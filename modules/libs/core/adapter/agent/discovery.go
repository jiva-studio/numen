// Package agent finds the programs a person already has installed on this machine.
package agent

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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
func IsClaudeInstalled() bool { return locate([]string{"claude"}, claudeSearchPlaces) != "" }

// IsAntigravityInstalled checks whether Antigravity CLI is available on this machine.
func IsAntigravityInstalled() bool {
	return locate([]string{"agy", "antigravity"}, antigravitySearchPlaces) != ""
}

// IsCodexInstalled checks whether OpenAI Codex CLI is available on this machine.
func IsCodexInstalled() bool { return locate([]string{"codex"}, codexSearchPlaces) != "" }

// FindClaude returns the path of the Claude Code CLI, or the bare name where it is nowhere.
func FindClaude() string { return findCommand("claude", []string{"claude"}, claudeSearchPlaces) }

// FindAntigravity returns the path of the Antigravity CLI, or the bare name where it is nowhere.
func FindAntigravity() string {
	return findCommand("agy", []string{"agy", "antigravity"}, antigravitySearchPlaces)
}

// FindCodex returns the path of the Codex CLI, or the bare name where it is nowhere.
func FindCodex() string { return findCommand("codex", []string{"codex"}, codexSearchPlaces) }

// findCommand returns the located program, or the bare name that says it is not installed.
func findCommand(bare string, names, places []string) string {
	if found := locate(names, places); found != "" {
		return found
	}
	return bare
}

// locate returns the first of names on the path, then the first of places that can run. Empty
// says none does.
func locate(names, places []string) string {
	for _, name := range names {
		if found, err := exec.LookPath(name); err == nil {
			return found
		}
	}
	return findInSearchPlaces(places)
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
			// gomutants:disable-next-line INTEGER_DECREMENT reason="filepath.Join cleans the doubled separator place[1:] leaves"
			place = filepath.Join(home, place[2:])
		}
		matches, err := filepath.Glob(place)
		if err != nil {
			continue
		}
		for _, match := range matches {
			if isRunnable(match) {
				return match
			}
		}
	}
	return ""
}

// isRunnable is a regular file with an execute bit, held under the name it was looked for by.
func isRunnable(path string) bool {
	about, err := os.Stat(path)
	if err != nil || !about.Mode().IsRegular() || about.Mode().Perm()&0o111 == 0 {
		return false
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		return false
	}
	name := filepath.Base(path)
	return slices.ContainsFunc(entries, func(e os.DirEntry) bool { return e.Name() == name })
}
