package container

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

const tokenPlugin = "org.example.courier"

func writeTokenPlugin(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "courier")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
		"id": "` + tokenPlugin + `",
		"name": "Courier",
		"version": "1.0.0",
		"targets": ["desktop.editor"],
		"settings": [{"key": "token", "type": "string", "title": "Token", "generate": "token"}]
	}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func loadTokenSettings(t *testing.T, cfg Config, plugins, path string) any {
	t.Helper()
	host := newPluginHost(t, "desktop.editor", plugins)
	t.Cleanup(func() {
		if err := host.Watch.TeardownAll(context.WithoutCancel(t.Context()), time.Millisecond); err != nil {
			t.Errorf("TeardownAll: %v", err)
		}
	})
	cfg.PluginHost = host
	selected, err := cfg.Settings()
	if err != nil {
		t.Fatal(err)
	}
	cfg.SetSettings(selected)

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var file map[string]any
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("numen.json is no object: %v\n%s", err, raw)
	}
	values, ok := file[tokenPlugin].(map[string]any)
	if !ok {
		t.Fatalf("numen.json holds no settings of %s:\n%s", tokenPlugin, raw)
	}
	return values["token"]
}

func TestLoadStoresAGeneratedTokenInNumenJSONAndASecondLoadKeepsIt(t *testing.T) {
	for name, body := range map[string]string{
		"an empty file":        `{}`,
		"a token stored empty": `{"` + tokenPlugin + `": {"token": ""}}`,
	} {
		t.Run(name, func(t *testing.T) {
			plugins := writeTokenPlugin(t)
			cfg, path := writeSettings(t, body)

			first := loadTokenSettings(t, cfg, plugins, path)
			token, ok := first.(string)
			if !ok || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(token) {
				t.Fatalf("numen.json holds the token %#v, want 64 lowercase hex characters", first)
			}

			if again := loadTokenSettings(t, cfg, plugins, path); again != token {
				t.Errorf("the next load left the token %#v, want %q", again, token)
			}
		})
	}
}

func TestNumenJSONKeepsATokenThePersonSet(t *testing.T) {
	const chosen = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	plugins := writeTokenPlugin(t)
	cfg, path := writeSettings(t, `{"`+tokenPlugin+`": {"token": "`+chosen+`"}}`)

	if got := loadTokenSettings(t, cfg, plugins, path); got != chosen {
		t.Errorf("numen.json holds the token %#v, want the one the person set", got)
	}
}
