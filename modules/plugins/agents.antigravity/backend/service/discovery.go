package service

import "github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun/programs"

var antigravityProgram = programs.Program{
	Names: []string{"agy", "antigravity"},
	Places: []string{
		"~/.gemini/antigravity-cli/bin/agy",
		"~/.antigravity/bin/agy",
		"/etc/profiles/per-user/*/bin/antigravity",
		"/etc/profiles/per-user/*/bin/agy",
	},
}

// IsAntigravityInstalled checks whether Antigravity CLI is available on this machine.
func IsAntigravityInstalled() bool {
	_, ok := programs.Discover(antigravityProgram)
	return ok
}

// FindAntigravity returns the path of the Antigravity CLI, or the bare name where not found.
func FindAntigravity() string {
	path, _ := programs.Discover(antigravityProgram)
	return path
}
