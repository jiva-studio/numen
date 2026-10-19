package main

import "github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun"

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

// IsClaudeInstalled checks whether Claude Code CLI is available on this machine; it answers completions.
func IsClaudeInstalled() bool { return agentrun.Locate([]string{"claude"}, claudeSearchPlaces) != "" }

// FindClaude returns the path of the Claude Code CLI, or the bare name where not found.
func FindClaude() string { return findCommand("claude", []string{"claude"}, claudeSearchPlaces) }

// IsAntigravityInstalled checks whether Antigravity CLI is available on this machine.
func IsAntigravityInstalled() bool {
	return agentrun.Locate([]string{"agy", "antigravity"}, antigravitySearchPlaces) != ""
}

// IsCodexInstalled checks whether OpenAI Codex CLI is available on this machine.
func IsCodexInstalled() bool { return agentrun.Locate([]string{"codex"}, codexSearchPlaces) != "" }

// IsCursorInstalled checks whether Cursor CLI is available on this machine.
func IsCursorInstalled() bool { return agentrun.Locate([]string{"cursor"}, cursorSearchPlaces) != "" }

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

// findCommand returns the located program, or the bare name that says it is not installed.
func findCommand(bare string, names, places []string) string {
	if found := agentrun.Locate(names, places); found != "" {
		return found
	}
	return bare
}
