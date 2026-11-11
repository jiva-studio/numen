package plugin

import (
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

const tokenPluginID = "org.example.relay"

// tokenPattern is 32 random bytes in lowercase hex.
var tokenPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// tokenWrites records what the host hands to its persist handler.
type tokenWrites struct {
	mu     sync.Mutex
	values map[string][]any
}

func (w *tokenWrites) add(pluginID, key string, value any) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.values == nil {
		w.values = map[string][]any{}
	}
	w.values[pluginID+"/"+key] = append(w.values[pluginID+"/"+key], value)
	return nil
}

func (w *tokenWrites) get(pluginID, key string) []any {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]any(nil), w.values[pluginID+"/"+key]...)
}

func newTokenHost(t *testing.T) *Host {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "relay")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
		"id": "` + tokenPluginID + `",
		"name": "Relay",
		"version": "1.0.0",
		"targets": ["desktop.editor"],
		"settings": [
			{"key": "token", "type": "string", "title": "Token", "generate": "token"},
			{"key": "secondToken", "type": "string", "title": "Second token", "generate": "token"},
			{"key": "label", "type": "string", "title": "Label"}
		]
	}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	host := openHost(testsupport.GetMachine(), "desktop.editor", root)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}
	return host
}

func loadTokenHost(host *Host, stored map[string]map[string]any, writes *tokenWrites) {
	host.Settings.SetSettings(stored)
	host.Settings.SetPersistHandler(writes.add)
}

func TestLoadPersistsOneGeneratedTokenForAnEnabledPlugin(t *testing.T) {
	for name, stored := range map[string]map[string]map[string]any{
		"no settings stored":        {},
		"the plugin stores none":    {tokenPluginID: {}},
		"the token is stored empty": {tokenPluginID: {"token": "", "secondToken": ""}},
		"the plugin is enabled":     {tokenPluginID: {"enabled": true}},
	} {
		t.Run(name, func(t *testing.T) {
			host := newTokenHost(t)
			writes := &tokenWrites{}
			loadTokenHost(host, stored, writes)

			got := writes.get(tokenPluginID, "token")
			if len(got) != 1 {
				t.Fatalf("token was written %d times (%v), want once", len(got), got)
			}
			token, ok := got[0].(string)
			if !ok || !tokenPattern.MatchString(token) {
				t.Fatalf("the written token is %#v, want 64 lowercase hex characters", got[0])
			}
			if inForce := host.Settings.GetPluginSettings(tokenPluginID)["token"]; inForce != token {
				t.Errorf("the token in force is %#v, want the written %q", inForce, token)
			}
			second := writes.get(tokenPluginID, "secondToken")
			if len(second) != 1 || second[0] == token {
				t.Errorf("secondToken was written as %v, want one value of its own", second)
			}
			if label := writes.get(tokenPluginID, "label"); len(label) != 0 {
				t.Errorf("a plain setting was written: %v", label)
			}
		})
	}
}

func TestASecondLoadKeepsTheGeneratedToken(t *testing.T) {
	host := newTokenHost(t)
	writes := &tokenWrites{}
	loadTokenHost(host, map[string]map[string]any{}, writes)
	first := writes.get(tokenPluginID, "token")
	if len(first) != 1 {
		t.Fatalf("the first load wrote token %v, want once", first)
	}

	loadTokenHost(host, map[string]map[string]any{tokenPluginID: {"token": first[0]}}, writes)
	if got := writes.get(tokenPluginID, "token"); len(got) != 1 {
		t.Errorf("token was written %v after the second load, want only the first value", got)
	}
	if inForce := host.Settings.GetPluginSettings(tokenPluginID)["token"]; inForce != first[0] {
		t.Errorf("the token in force is %#v, want %#v", inForce, first[0])
	}

	restarted := newTokenHost(t)
	again := &tokenWrites{}
	loadTokenHost(restarted, map[string]map[string]any{tokenPluginID: {"token": first[0], "secondToken": "f00dfeed"}}, again)
	if got := again.get(tokenPluginID, "token"); len(got) != 0 {
		t.Errorf("a host started on the stored token wrote %v", got)
	}
	if inForce := restarted.Settings.GetPluginSettings(tokenPluginID)["token"]; inForce != first[0] {
		t.Errorf("the token in force after a restart is %#v, want %#v", inForce, first[0])
	}
}

func TestLoadKeepsATokenThePersonSet(t *testing.T) {
	const chosen = "c0ffee00c0ffee00c0ffee00c0ffee00c0ffee00c0ffee00c0ffee00c0ffee00"
	host := newTokenHost(t)
	writes := &tokenWrites{}
	loadTokenHost(host, map[string]map[string]any{tokenPluginID: {"token": chosen, "secondToken": "my-own-phrase"}}, writes)

	if got := writes.get(tokenPluginID, "token"); len(got) != 0 {
		t.Errorf("token was written %v over the value the person set", got)
	}
	if got := writes.get(tokenPluginID, "secondToken"); len(got) != 0 {
		t.Errorf("secondToken was written %v over the value the person set", got)
	}
	settings := host.Settings.GetPluginSettings(tokenPluginID)
	if settings["token"] != chosen || settings["secondToken"] != "my-own-phrase" {
		t.Errorf("settings in force = %v, want the values the person set", settings)
	}
}

func TestLoadGeneratesNoTokenForADisabledPlugin(t *testing.T) {
	host := newTokenHost(t)
	writes := &tokenWrites{}
	loadTokenHost(host, map[string]map[string]any{tokenPluginID: {"enabled": false}}, writes)

	if got := writes.get(tokenPluginID, "token"); len(got) != 0 {
		t.Errorf("a disabled plugin had its token written: %v", got)
	}
	if _, ok := host.Settings.GetPluginSettings(tokenPluginID)["token"]; ok {
		t.Errorf("a disabled plugin holds a token: %v", host.Settings.GetPluginSettings(tokenPluginID))
	}
}
