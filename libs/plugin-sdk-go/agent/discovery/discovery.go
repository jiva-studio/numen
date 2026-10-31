// Package discovery finds the command-line program of an agent on the machine.
package discovery

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
)

// Program describes where a program can be found.
type Program struct {
	// Names are the executable names searched for on PATH, in order.
	Names []string
	// Places are the paths searched when PATH holds none of the names. A leading "~/" is the home folder.
	Places []string
	// Fallback is the name returned where the program is not found.
	Fallback string
}

// Find returns the path of the program: customPath when it is runnable, else the first match on
// PATH, else the first runnable place, else the fallback name.
func (p Program) Find(customPath ...string) string {
	if len(customPath) > 0 && customPath[0] != "" && agentrun.IsRunnable(customPath[0]) {
		return customPath[0]
	}
	for _, name := range p.Names {
		if found, err := exec.LookPath(name); err == nil {
			return found
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	for _, place := range p.Places {
		resolved := place
		if strings.HasPrefix(place, "~/") && home != "" {
			resolved = filepath.Join(home, place[2:])
		}
		if agentrun.IsRunnable(resolved) {
			return resolved
		}
	}
	return p.Fallback
}

// IsInstalled reports whether customPath, when given, or else the program found by Find is runnable.
func (p Program) IsInstalled(customPath ...string) bool {
	if len(customPath) > 0 && customPath[0] != "" {
		return agentrun.IsRunnable(customPath[0])
	}
	return agentrun.IsRunnable(p.Find())
}

// GetCommand returns the command line the installation set for the named program, or nil.
func GetCommand(commands []*agentrun.AgentProgramCommand, name string) []string {
	for _, one := range commands {
		if one.GetProgram() == name {
			return one.GetCommand()
		}
	}
	return nil
}
