package container_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
)

const (
	courierPlugin = "org.example.courier"
	courierToken  = "5ca1ab1e5ca1ab1e5ca1ab1e5ca1ab1e5ca1ab1e5ca1ab1e5ca1ab1e5ca1ab1e"
)

func openDisabledCourier(t *testing.T) (container.Config, string, *recordingHost, func(string) map[string]any) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "courier")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
		"id": "` + courierPlugin + `",
		"name": "Courier",
		"version": "1.0.0",
		"targets": ["desktop.editor"],
		"settings": [{"key": "token", "type": "string", "title": "Token", "generate": "token"}]
	}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "numen.json")
	if err := os.WriteFile(path, []byte(`{"`+courierPlugin+`":{"enabled":false}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	pluginHost := pluginhost.Open(t, "desktop.editor", root)
	lent := container.AdaptPluginHost((*container.PluginRoles)(pluginHost))
	host := &recordingHost{apply: lent.ApplySetting, SettingsPush: lent.Configuration.SettingsPush}
	lent.ApplySetting, lent.Configuration.SettingsPush = host.applySetting, host
	cfg := container.NewConfig(container.Config{
		PluginHost: lent,
		Paths:      container.Paths{SettingsPath: path},
	})
	selected, err := cfg.Settings()
	if err != nil {
		t.Fatal(err)
	}
	cfg = cfg.SetSettings(selected)
	if token, ok := pluginHost.Settings.GetPluginSettings(courierPlugin)["token"]; ok {
		t.Fatalf("the disabled plugin holds the token %v after load", token)
	}
	return cfg, path, host, pluginHost.Settings.GetPluginSettings
}

func readCourierToken(t *testing.T, path string) any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var file map[string]any
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("numen.json is no object: %v\n%s", err, raw)
	}
	values, ok := file[courierPlugin].(map[string]any)
	if !ok {
		t.Fatalf("numen.json holds no settings of %s:\n%s", courierPlugin, raw)
	}
	return values["token"]
}

func TestEnablingAPluginTogetherWithATokenKeepsThePersonsToken(t *testing.T) {
	for name, write := range map[string]func(t *testing.T, cfg container.Config, path string){
		"one write of both settings": func(t *testing.T, cfg container.Config, _ string) {
			t.Helper()
			if err := container.GetSettingsWriter(cfg)([]settings.Setting{
				{Path: []string{courierPlugin, "enabled"}, JSON: `true`},
				{Path: []string{courierPlugin, "token"}, JSON: `"` + courierToken + `"`},
			}); err != nil {
				t.Fatal(err)
			}
		},
		"an edit of numen.json by hand": func(t *testing.T, _ container.Config, path string) {
			t.Helper()
			body := `{"` + courierPlugin + `":{"enabled":true,"token":"` + courierToken + `"}}`
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			cfg, path, host, getPluginSettings := openDisabledCourier(t)
			reader := listen(t, cfg)

			write(t, cfg, path)

			reader.await(t)
			waitFor(t, func() bool {
				updates := host.listUpdates()
				return slices.Contains(updates, courierPlugin+".enabled") && slices.Contains(updates, courierPlugin+".token")
			})
			deadline := time.After(quiet)
			for isQuiet := false; !isQuiet; {
				select {
				case batch := <-reader.changes:
					for _, one := range batch {
						if slices.Equal(one.Path, []string{courierPlugin, "token"}) && one.JSON != `"`+courierToken+`"` {
							t.Errorf("a later change wrote the token %s over the one the person set", one.JSON)
						}
					}
				case <-deadline:
					isQuiet = true
				}
			}

			if got := readCourierToken(t, path); got != courierToken {
				t.Errorf("numen.json holds the token %#v, want the one the person set", got)
			}
			if got := getPluginSettings(courierPlugin)["token"]; got != courierToken {
				t.Errorf("the host holds the token %#v, want the one the person set", got)
			}
		})
	}
}
