package acp

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

var antigravityPlaces = []string{
	"~/.local/bin/agy",
	"~/.local/bin/antigravity",
	"/etc/profiles/per-user/*/bin/antigravity",
	"/etc/profiles/per-user/*/bin/agy",
	"~/.gemini/antigravity-cli/bin/agy",
	"~/.antigravity/bin/agy",
	"/opt/homebrew/bin/agy",
	"/usr/local/bin/agy",
	"/run/current-system/sw/bin/agy",
	"/run/current-system/sw/bin/antigravity",
	"~/.nix-profile/bin/agy",
	"~/.nix-profile/bin/antigravity",
}

var codexPlaces = []string{
	"~/.local/bin/codex",
	"~/.codex/bin/codex",
	"~/.npm-global/bin/codex",
	"~/.bun/bin/codex",
	"/opt/homebrew/bin/codex",
	"/usr/local/bin/codex",
	"/run/current-system/sw/bin/codex",
}

// FindAntigravity finds the Antigravity CLI binary.
func FindAntigravity() string {
	if named, err := exec.LookPath("agy"); err == nil {
		return named
	}
	if named, err := exec.LookPath("antigravity"); err == nil {
		return named
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	if found := findInPlaces(home, antigravityPlaces); found != "" {
		return found
	}
	return "agy"
}

// FindCodex finds the OpenAI Codex CLI binary.
func FindCodex() string {
	if named, err := exec.LookPath("codex"); err == nil {
		return named
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	if found := findInPlaces(home, codexPlaces); found != "" {
		return found
	}
	return "codex"
}

func findInPlaces(home string, places []string) string {
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
