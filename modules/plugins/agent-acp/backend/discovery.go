package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

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

var cursorSearchPlaces = []string{
	"~/.local/bin/cursor",
	"/usr/local/bin/cursor",
	"/opt/homebrew/bin/cursor",
	"~/.nix-profile/bin/cursor",
	"/run/current-system/sw/bin/cursor",
	"~/Applications/Cursor.app/Contents/MacOS/Cursor",
	"/Applications/Cursor.app/Contents/MacOS/Cursor",
	"~/.cursor/bin/cursor",
	"/usr/bin/cursor",
}

// IsAntigravityInstalled checks whether Antigravity CLI is available on this machine.
func IsAntigravityInstalled() bool {
	return locate([]string{"agy", "antigravity"}, antigravitySearchPlaces) != ""
}

// IsCodexInstalled checks whether OpenAI Codex CLI is available on this machine.
func IsCodexInstalled() bool { return locate([]string{"codex"}, codexSearchPlaces) != "" }

// IsCursorInstalled checks whether Cursor CLI is available on this machine.
func IsCursorInstalled() bool { return locate([]string{"cursor"}, cursorSearchPlaces) != "" }

// FindAntigravity returns the path of the Antigravity CLI, or the bare name where not found.
func FindAntigravity() string {
	return findCommand("agy", []string{"agy", "antigravity"}, antigravitySearchPlaces)
}

// FindCodex returns the path of the Codex CLI, or the bare name where not found.
func FindCodex() string { return findCommand("codex", []string{"codex"}, codexSearchPlaces) }

// FindCursor returns the path of the Cursor CLI, or the bare name where not found.
func FindCursor() string { return findCommand("cursor", []string{"cursor"}, cursorSearchPlaces) }

// GetInstalledAgents returns the identifiers of all external agents present on this machine.
func GetInstalledAgents() []string {
	var agents []string
	if IsAntigravityInstalled() {
		agents = append(agents, "antigravity")
	}
	if IsCodexInstalled() {
		agents = append(agents, "codex")
	}
	if IsCursorInstalled() {
		agents = append(agents, "cursor")
	}
	return agents
}

func findCommand(bare string, names, places []string) string {
	if found := locate(names, places); found != "" {
		return found
	}
	return bare
}

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
