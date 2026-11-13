// Package executable finds the Claude Code program on the machine.
package executable

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

func getSearchPlaces() []string {
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

// Finder looks for the program in the environment, on the path and in the places it is installed to.
type Finder struct{}

// Find returns the command of the program: the custom path when it is runnable, else the command the
// environment names, else the program on the path, else a place it is installed to.
func (Finder) Find(customPath string) string {
	return findIn(getSearchPlaces(), customPath)
}

func findIn(places []string, customPath string) string {
	if customPath != "" && isRunnable(customPath) {
		return customPath
	}
	if cmd := os.Getenv("NUMEN_CLAUDE_COMMAND"); cmd != "" {
		return cmd
	}
	if cmd := os.Getenv("CLAUDE_COMMAND"); cmd != "" {
		return cmd
	}
	return locateExecutablePath([]string{"claude"}, places)
}

func locateExecutablePath(names, places []string) string {
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
			if isRunnable(match) {
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
