package service

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

type program struct {
	Names  []string
	Places []string
}

var codexProgram = program{
	Names: []string{"codex"},
	Places: []string{
		"~/.codex/bin/codex",
		"~/.npm-global/bin/codex",
		"~/.bun/bin/codex",
	},
}

// IsCodexInstalled checks whether OpenAI Codex CLI is available on this machine.
func IsCodexInstalled() bool {
	_, ok := discover(codexProgram)
	return ok
}

// FindCodex returns the path of the Codex CLI, or the bare name where not found.
func FindCodex() string {
	path, _ := discover(codexProgram)
	return path
}

func programCommand(commands []*numenv1.AgentProgramCommand, name string) []string {
	for _, one := range commands {
		if one.GetProgram() == name {
			return one.GetCommand()
		}
	}
	return nil
}

func discover(p program) (string, bool) {
	var places []string
	for _, name := range p.Names {
		places = append(places, "~/.local/bin/"+name)
	}
	places = append(places, p.Places...)
	for _, name := range p.Names {
		places = append(places,
			"/opt/homebrew/bin/"+name,
			"/usr/local/bin/"+name,
			"~/.nix-profile/bin/"+name,
			"/run/current-system/sw/bin/"+name,
		)
	}
	if found := locate(p.Names, places); found != "" {
		return found, true
	}
	if len(p.Names) > 0 {
		return p.Names[0], false
	}
	return "", false
}

func locate(names, places []string) string {
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
		ext := strings.ToLower(filepath.Ext(path))
		return slices.Contains([]string{".exe", ".cmd", ".bat"}, ext)
	}
	return info.Mode()&0o111 != 0
}
