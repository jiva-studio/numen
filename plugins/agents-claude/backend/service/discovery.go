package service

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
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

// findClaude returns the custom path if specified and runnable, else the command the environment
// names, else the Claude Code CLI where it is installed. Empty is a program nowhere to be found.
func findClaude(customPath ...string) string {
	if len(customPath) > 0 && customPath[0] != "" {
		if agentrun.IsRunnable(customPath[0]) {
			return customPath[0]
		}
	}
	if cmd := os.Getenv("NUMEN_CLAUDE_COMMAND"); cmd != "" {
		return cmd
	}
	if cmd := os.Getenv("CLAUDE_COMMAND"); cmd != "" {
		return cmd
	}
	return locateExecutable([]string{"claude"}, claudeSearchPlaces)
}

func locateExecutable(names, places []string) string {
	for _, name := range names {
		if found, err := exec.LookPath(name); err == nil {
			return found
		}
	}
	return findInPlaces(places)
}

func findInPlaces(places []string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	for _, place := range places {
		if strings.HasPrefix(place, "~/") {
			if home == "" {
				continue
			}
			place = filepath.Join(escapeGlob(home), place[2:])
		}
		matches, err := filepath.Glob(place)
		if err != nil {
			continue
		}
		for _, match := range matches {
			if agentrun.IsRunnable(match) {
				return match
			}
		}
	}
	return ""
}

func escapeGlob(path string) string {
	var pattern strings.Builder
	for _, r := range path {
		switch {
		case r == '*' || r == '?' || r == '[':
			pattern.WriteString("[" + string(r) + "]")
		case r == '\\' && runtime.GOOS != "windows":
			pattern.WriteString(`\\`)
		default:
			pattern.WriteRune(r)
		}
	}
	return pattern.String()
}
