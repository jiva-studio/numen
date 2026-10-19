package main

import "github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun"

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

var codexACPSearchPlaces = []string{
	"~/.local/bin/codex-acp",
	"~/.npm-global/bin/codex-acp",
	"~/.bun/bin/codex-acp",
	"/opt/homebrew/bin/codex-acp",
	"/usr/local/bin/codex-acp",
	"~/.nix-profile/bin/codex-acp",
	"/run/current-system/sw/bin/codex-acp",
}

var cursorAgentSearchPlaces = []string{
	"~/.local/bin/cursor-agent",
	"/opt/homebrew/bin/cursor-agent",
	"/usr/local/bin/cursor-agent",
	"~/.nix-profile/bin/cursor-agent",
	"/run/current-system/sw/bin/cursor-agent",
}

// acpAgent is an installed agent that serves the Agent Client Protocol over its standard streams.
type acpAgent struct {
	// Program is what a completion model names the agent by.
	Program string
	Title   string
	// Command starts the agent's ACP server.
	Command []string
}

// GetACPAgents returns the installed ACP agents, the one a completion takes by default first:
// Codex through its codex-acp server, and the Cursor agent through its acp command.
func GetACPAgents() []acpAgent {
	var agents []acpAgent
	if found := agentrun.Locate([]string{"codex-acp"}, codexACPSearchPlaces); found != "" {
		agents = append(agents, acpAgent{Program: "codex", Title: "OpenAI Codex", Command: []string{found}})
	}
	if found := agentrun.Locate([]string{"cursor-agent"}, cursorAgentSearchPlaces); found != "" {
		agents = append(agents, acpAgent{Program: "cursor", Title: "Cursor", Command: []string{found, "acp"}})
	}
	return agents
}
