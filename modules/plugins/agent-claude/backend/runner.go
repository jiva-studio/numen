package main

import (
	"encoding/json"
	"fmt"
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

func findClaude() string {
	if cmd := os.Getenv("NUMEN_CLAUDE_COMMAND"); cmd != "" {
		return cmd
	}
	if cmd := os.Getenv("CLAUDE_COMMAND"); cmd != "" {
		return cmd
	}
	if found, err := exec.LookPath("claude"); err == nil {
		return found
	}
	return findInSearchPlaces(claudeSearchPlaces)
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
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return false
	}
	return true
}

func writeMcpConfig(url, token string) (string, error) {
	if url == "" || token == "" {
		return "", nil
	}
	type server struct {
		Type    string            `json:"type"`
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
	}
	config := struct {
		Servers map[string]server `json:"mcpServers"`
	}{
		Servers: map[string]server{
			"numen": {
				Type:    "http",
				URL:     url,
				Headers: map[string]string{"Authorization": "Bearer " + token},
			},
		},
	}
	data, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("marshal mcp config: %w", err)
	}

	file, err := os.CreateTemp("", "numen-tools-*.json")
	if err != nil {
		return "", fmt.Errorf("create temp config: %w", err)
	}
	defer file.Close()

	if err := file.Chmod(0o600); err != nil {
		_ = os.Remove(file.Name())
		return "", fmt.Errorf("chmod config: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		_ = os.Remove(file.Name())
		return "", fmt.Errorf("write config: %w", err)
	}
	return file.Name(), nil
}

func buildClaudeArguments(
	instructions, model, configuration, session string,
	turns int,
	tools []string,
	mayReadHooks bool,
) []string {
	if turns <= 0 {
		turns = 100
	}

	args := []string{
		"-p",
		"--output-format", "stream-json",
		"--verbose",
		"--include-partial-messages",
		"--permission-mode", "dontAsk",
		"--max-turns", fmt.Sprint(turns),
	}

	if configuration != "" {
		args = append(args, "--strict-mcp-config", "--mcp-config", configuration)
	}

	args = append(args, "--tools", "WebSearch")

	var allowed []string
	allowed = append(allowed, "WebSearch")
	for _, t := range tools {
		name := t
		if !strings.HasPrefix(name, "mcp__numen__") {
			name = "mcp__numen__" + name
		}
		allowed = append(allowed, name)
	}
	slices.Sort(allowed)
	args = append(args, "--allowedTools", strings.Join(allowed, ","))

	// A vault's own hooks and settings are never read. The person's own are read only when the
	// setting allows it.
	sources := ""
	if mayReadHooks {
		sources = "user"
	}
	args = append(args, "--setting-sources", sources)

	if instructions != "" {
		args = append(args, "--append-system-prompt", instructions)
	}
	if model != "" {
		args = append(args, "--model", model)
	}
	if session != "" {
		args = append(args, "--resume", session)
	}
	return args
}
