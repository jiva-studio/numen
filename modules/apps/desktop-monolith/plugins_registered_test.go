package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// absentPlugins are the ids of the plugins with a backend that this binary does
// not link in.
var absentPlugins = []string{"md.numen.agent-acp", "md.numen.agent-antigravity", "md.numen.agent-codex"}

// getUnaccountedPlugins lists the plugins with a backend that are neither
// registered nor listed as absent, and the absent ones that are registered or
// have no plugin folder.
func getUnaccountedPlugins(backends, registered, absent []string) []string {
	var unaccounted []string
	for i, id := range backends {
		if slices.Contains(backends[:i], id) {
			unaccounted = append(unaccounted, id+" is declared by more than one backend")
		}
		if !slices.Contains(registered, id) && !slices.Contains(absent, id) {
			unaccounted = append(unaccounted, id+" is neither registered nor listed as absent")
		}
	}
	for _, id := range absent {
		switch {
		case slices.Contains(registered, id):
			unaccounted = append(unaccounted, id+" is listed as absent and registered")
		case !slices.Contains(backends, id):
			unaccounted = append(unaccounted, id+" is listed as absent and has no backend")
		}
	}
	return unaccounted
}

// What getUnaccountedPlugins says is the whole of the rule, so it is asked
// directly.
func TestWhatThePluginRegistrationRuleRefuses(t *testing.T) {
	backends := []string{"a", "b", "c"}
	if got := getUnaccountedPlugins(backends, []string{"a", "b"}, []string{"c"}); len(got) != 0 {
		t.Fatalf("an accounted set is refused: %v", got)
	}
	for name, got := range map[string][]string{
		"a plugin neither registered nor absent": getUnaccountedPlugins(backends, []string{"a", "b"}, nil),
		"an absent plugin that is registered":    getUnaccountedPlugins(backends, []string{"a", "b", "c"}, []string{"c"}),
		"an id declared twice":                   getUnaccountedPlugins([]string{"a", "a"}, []string{"a"}, nil),
		"an absent plugin that does not exist":   getUnaccountedPlugins(backends, []string{"a", "b", "c"}, []string{"d"}),
	} {
		if len(got) == 0 {
			t.Errorf("%s is not refused", name)
		}
	}
}

// Every plugin with a backend is registered by the composition root or listed
// there as absent.
func TestEveryPluginWithABackendIsRegisteredOrAbsent(t *testing.T) {
	manifests, err := filepath.Glob(filepath.Join("..", "..", "plugins", "*", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(manifests) < 2 {
		t.Fatalf("%d manifests found: a rule checked against nothing", len(manifests))
	}
	var backends []string
	for _, path := range manifests {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var declared struct {
			ID      string `json:"id"`
			Backend struct {
				Entrypoint string `json:"entrypoint"`
			} `json:"backend"`
		}
		if err := json.Unmarshal(data, &declared); err != nil {
			t.Fatalf("%s does not parse: %v", path, err)
		}
		if declared.Backend.Entrypoint != "" {
			backends = append(backends, declared.ID)
		}
	}

	var registered []string
	registerPlugins(func(manifest []byte, _ serveFunc) {
		var declared struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(manifest, &declared); err != nil {
			t.Fatalf("a manifest does not parse: %v", err)
		}
		registered = append(registered, declared.ID)
	})

	for _, issue := range getUnaccountedPlugins(backends, registered, absentPlugins) {
		t.Error(issue)
	}
}
