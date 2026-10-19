package service

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

// IsAntigravityInstalled checks whether Antigravity CLI is available on this machine.
func IsAntigravityInstalled() bool {
	return agentrun.Locate([]string{"agy", "antigravity"}, antigravitySearchPlaces) != ""
}

// FindAntigravity returns the path of the Antigravity CLI, or the bare name where not found.
func FindAntigravity() string {
	if found := agentrun.Locate([]string{"agy", "antigravity"}, antigravitySearchPlaces); found != "" {
		return found
	}
	return "agy"
}
