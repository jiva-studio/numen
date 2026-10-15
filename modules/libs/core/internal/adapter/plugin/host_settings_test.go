package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

func TestHostSettingsOverlayAndPersistence(t *testing.T) {
	tmp := t.TempDir()
	pDir := filepath.Join(tmp, "sample")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	manifest := `{
		"id": "md.numen.sample",
		"name": "Sample Plugin",
		"version": "1.0.0",
		"description": "Sample description",
		"capabilities": ["sample"],
		"targets": ["desktop.editor"],
		"settings": [
			{"key": "enabled", "type": "boolean", "title": "Enabled", "default": true, "palette": true},
			{"key": "count", "type": "number", "title": "Count", "default": 5},
			{"key": "mode", "type": "enum", "title": "Mode", "default": "fast", "options": ["fast", "accurate"]}
		]
	}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest failed: %v", err)
	}

	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("discover failed: %v", err)
	}

	// 1. Initial settings should match defaults
	initial := host.GetPluginSettings("md.numen.sample")
	if initial["enabled"] != true {
		t.Errorf("expected default enabled=true, got %v", initial["enabled"])
	}
	if initial["count"] != float64(5) {
		t.Errorf("expected default count=5, got %v", initial["count"])
	}
	if initial["mode"] != "fast" {
		t.Errorf("expected default mode=fast, got %v", initial["mode"])
	}

	// 2. SetSettings from configuration file overlay
	host.SetSettings(map[string]map[string]any{
		"md.numen.sample": {
			"count": 10,
		},
	})
	overlaid := host.GetPluginSettings("md.numen.sample")
	if overlaid["enabled"] != true {
		t.Errorf("expected enabled=true from default, got %v", overlaid["enabled"])
	}
	if overlaid["count"] != 10 {
		t.Errorf("expected overlaid count=10, got %v", overlaid["count"])
	}

	// 3. SetPluginSetting with persist handler
	var persistedPlugin, persistedKey string
	var persistedVal any
	host.SetPersistHandler(func(pID string, key string, val any) error {
		persistedPlugin = pID
		persistedKey = key
		persistedVal = val
		return nil
	})

	ctx := t.Context()
	if err := host.SetPluginSetting(ctx, "md.numen.sample", "mode", "accurate"); err != nil {
		t.Fatalf("SetPluginSetting failed: %v", err)
	}
	if persistedPlugin != "md.numen.sample" || persistedKey != "mode" || persistedVal != "accurate" {
		t.Errorf("persist handler called with (%s, %s, %v), want (md.numen.sample, mode, accurate)",
			persistedPlugin, persistedKey, persistedVal)
	}

	updated := host.GetPluginSettings("md.numen.sample")
	if updated["mode"] != "accurate" {
		t.Errorf("expected updated mode=accurate, got %v", updated["mode"])
	}

	// 4. Config JSON serialization
	cfgJSON, err := host.GetPluginConfigJSON("md.numen.sample")
	if err != nil {
		t.Fatalf("GetPluginConfigJSON failed: %v", err)
	}
	if cfgJSON == "" {
		t.Fatalf("expected non-empty config json")
	}

	// 5. PluginServiceServer.GetPlugins
	server := NewPluginServiceServer(host)
	resp, err := server.GetPlugins(ctx, connect.NewRequest(&v1.GetPluginsRequest{}))
	if err != nil {
		t.Fatalf("GetPlugins failed: %v", err)
	}
	if len(resp.Msg.Plugins) != 1 {
		t.Fatalf("expected 1 plugin info, got %d", len(resp.Msg.Plugins))
	}
	pInfo := resp.Msg.Plugins[0]
	if pInfo.Id != "md.numen.sample" {
		t.Errorf("expected id md.numen.sample, got %s", pInfo.Id)
	}
	if len(pInfo.Settings) != 3 {
		t.Fatalf("expected 3 settings in info, got %d", len(pInfo.Settings))
	}
	if pInfo.Settings[0].Key != "enabled" || !pInfo.Settings[0].Palette {
		t.Errorf("setting 0: expected key=enabled and palette=true")
	}

	// 6. Host.GetPaletteCommands
	palCmds := host.GetPaletteCommands()
	if len(palCmds) != 1 {
		t.Fatalf("expected 1 palette command, got %d", len(palCmds))
	}
	if palCmds[0].ID != "plugin-setting:md.numen.sample:enabled" {
		t.Errorf("got palette command ID %s, want plugin-setting:md.numen.sample:enabled", palCmds[0].ID)
	}

	// 7. Host.TogglePluginSetting
	newVal, err := host.TogglePluginSetting(ctx, "md.numen.sample", "enabled")
	if err != nil {
		t.Fatalf("TogglePluginSetting failed: %v", err)
	}
	if newVal != false {
		t.Errorf("expected toggled value false, got %v", newVal)
	}
	toggledSettings := host.GetPluginSettings("md.numen.sample")
	if toggledSettings["enabled"] != false {
		t.Errorf("expected updated setting enabled=false, got %v", toggledSettings["enabled"])
	}
}
