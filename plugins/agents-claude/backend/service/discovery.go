package service

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
)

func getClaudeSearchPlaces() []string {
	return []string{
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
}

func findClaude(customPath ...string) string {
	return findClaudeIn(getClaudeSearchPlaces(), customPath...)
}

func findClaudeIn(places []string, customPath ...string) string {
	if len(customPath) > 0 && agentrun.IsRunnable(customPath[0]) {
		return customPath[0]
	}
	if cmd := os.Getenv("NUMEN_CLAUDE_COMMAND"); cmd != "" {
		return cmd
	}
	if cmd := os.Getenv("CLAUDE_COMMAND"); cmd != "" {
		return cmd
	}
	return locateExecutable([]string{"claude"}, places)
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
		if after, ok := strings.CutPrefix(place, "~/"); ok {
			if home == "" {
				continue
			}
			place = filepath.Join(escapeGlob(home), after)
		}
		matches, err := filepath.Glob(place)
		// gomutants:disable-next-line BRANCH_IF reason="filepath.Glob returns nil matches on error so the loop below performs zero iterations."
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
