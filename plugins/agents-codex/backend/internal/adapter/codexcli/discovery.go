// Package codexcli reaches the Codex CLI and its files on this machine: it finds the program, lists the
// models it offers, reads its configuration and builds the command lines of a run and starts the program in a process group of its own.
package codexcli

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

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
		return isRunnable(customPath[0])
	}
	p := FindCodex()
	return isRunnable(p)
}

// FindCodex returns the path of the Codex CLI, or the bare name where not found.
func FindCodex(customPath ...string) string {
	return findCodexIn(getCodexPlaces(), customPath...)
}

func findCodexIn(places []string, customPath ...string) string {
	if len(customPath) > 0 && isRunnable(customPath[0]) {
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
		if isRunnable(resolved) {
			return resolved
		}
	}
	return "codex"
}

// Installation checks for the Codex CLI on this machine.
type Installation struct{}

// IsInstalled reports whether the CLI is runnable, at path when given, or where it is found.
func (Installation) IsInstalled(path string) bool {
	return IsCodexInstalled(path)
}

// Find returns the program: the custom path when it is runnable, else the Codex found on this machine,
// else none.
func (Installation) Find(customPath string) string {
	if !IsCodexInstalled(customPath) {
		return ""
	}
	return FindCodex(customPath)
}

func isRunnable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return slices.Contains([]string{".exe", ".cmd", ".bat"}, strings.ToLower(filepath.Ext(path)))
	}
	return info.Mode()&0o111 != 0
}
