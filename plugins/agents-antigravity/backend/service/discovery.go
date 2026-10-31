package service

import "github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/discovery"

// antigravityProgram is where the Antigravity CLI is searched for.
var antigravityProgram = discovery.Program{
	Names: []string{"agy", "antigravity"},
	Places: []string{
		"~/.local/bin/antigravity",
		"~/.local/bin/agy",
		"~/.antigravity/bin/antigravity",
		"/opt/homebrew/bin/antigravity",
		"/usr/local/bin/antigravity",
		"~/.nix-profile/bin/antigravity",
		"/run/current-system/sw/bin/antigravity",
	},
	Fallback: "agy",
}

// IsAntigravityInstalled checks whether Google Antigravity CLI is available on this machine.
func IsAntigravityInstalled(customPath ...string) bool {
	return antigravityProgram.IsInstalled(customPath...)
}

// FindAntigravity returns the path of the Antigravity CLI, or the bare name where not found.
func FindAntigravity(customPath ...string) string {
	return antigravityProgram.Find(customPath...)
}
