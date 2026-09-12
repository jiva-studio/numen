// The places the command line is looked for, and which file among them is the
// one to start.

package claudecode

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// places are where the command line is looked for when the path does not name
// it.
//
// An application opened from a desktop is given the system path alone, so every
// folder an installer writes to is named here. A leading ~ is this person's
// home, and a * is expanded.
//
// A mac carries programs inside application bundles, and the bundle the
// command line's installer leaves there holds a link to it.
var places = []string{
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

// findCommand is the command line to start: the path first, then the places.
//
// The bare name is the answer when it is nowhere, and starting that says it is
// not installed.
func findCommand() string {
	if named, err := exec.LookPath("claude"); err == nil {
		return named
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	if found := found(home, places); found != "" {
		return found
	}
	return "claude"
}

// found is the first of places that is a program this machine can run. Empty
// says none of them is.
func found(home string, places []string) string {
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
			if runnable(match) {
				return match
			}
		}
	}
	return ""
}

// runnable is a file with an execute bit on it, held under the name it was
// looked for by.
//
// A mac filesystem answers to a name in any case, so the folder is asked which
// name it keeps.
func runnable(path string) bool {
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
