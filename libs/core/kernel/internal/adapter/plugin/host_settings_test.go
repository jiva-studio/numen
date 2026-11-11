package plugin_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"
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
		"capabilities": ["tools"],
		"targets": ["desktop.editor"],
		"settings": [
			{"key": "verbose", "type": "boolean", "title": "Verbose", "default": true, "palette": true},
			{"key": "count", "type": "number", "title": "Count", "default": 5},
			{"key": "mode", "type": "enum", "title": "Mode", "default": "fast", "options": ["fast", "accurate"]}
		]
	}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest failed: %v", err)
	}

	host := pluginhost.New(testsupport.GetMachine(), "desktop.editor", tmp)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatalf("discover failed: %v", err)
	}

	// 1. Initial settings should match defaults
	initial := host.Settings.GetPluginSettings("md.numen.sample")
	if initial["verbose"] != true {
		t.Errorf("expected default verbose=true, got %v", initial["verbose"])
	}
	if initial["count"] != float64(5) {
		t.Errorf("expected default count=5, got %v", initial["count"])
	}
	if initial["mode"] != "fast" {
		t.Errorf("expected default mode=fast, got %v", initial["mode"])
	}

	// 2. SetSettings from configuration file overlay
	host.Settings.SetSettings(map[string]map[string]any{
		"md.numen.sample": {
			"count": 10,
		},
	})
	overlaid := host.Settings.GetPluginSettings("md.numen.sample")
	if overlaid["verbose"] != true {
		t.Errorf("expected verbose=true from default, got %v", overlaid["verbose"])
	}
	if overlaid["count"] != 10 {
		t.Errorf("expected overlaid count=10, got %v", overlaid["count"])
	}

	// 3. A setting applied is in the settings the plugin has
	ctx := t.Context()
	if err := updateSetting(ctx, host, "md.numen.sample", "mode", "accurate"); err != nil {
		t.Fatalf("updating the setting failed: %v", err)
	}

	updated := host.Settings.GetPluginSettings("md.numen.sample")
	if updated["mode"] != "accurate" {
		t.Errorf("expected updated mode=accurate, got %v", updated["mode"])
	}

	// 4. Config JSON serialization
	cfgJSON, err := host.Settings.GetPluginConfigJSON("md.numen.sample")
	if err != nil {
		t.Fatalf("GetPluginConfigJSON failed: %v", err)
	}
	if cfgJSON == "" {
		t.Fatalf("expected non-empty config json")
	}

	// 5. Host.GetPluginInfos
	infos := host.Registry.GetPluginInfos()
	if len(infos) != 1 {
		t.Fatalf("expected 1 plugin info, got %d", len(infos))
	}
	pInfo := infos[0]
	if pInfo.ID != "md.numen.sample" {
		t.Errorf("expected id md.numen.sample, got %s", pInfo.ID)
	}
	if len(pInfo.Settings) != 3 {
		t.Fatalf("expected 3 settings in info, got %d", len(pInfo.Settings))
	}
	if pInfo.Settings[0].Key != "verbose" || !pInfo.Settings[0].Palette {
		t.Errorf("setting 0: expected key=verbose and palette=true")
	}

}

func TestPluginEnablementAndStatus(t *testing.T) {
	tmp := t.TempDir()
	pDir := filepath.Join(tmp, "sample")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	manifest := `{
		"id": "md.numen.toggleable",
		"name": "Toggleable Plugin",
		"version": "1.0.0",
		"description": "Can be toggled",
		"capabilities": ["tools"],
		"targets": ["desktop.editor"],
		"activation_events": ["on-edit"]
	}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest failed: %v", err)
	}

	host := pluginhost.New(testsupport.GetMachine(), "desktop.editor", tmp)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatalf("discover failed: %v", err)
	}

	pluginID := "md.numen.toggleable"

	// 1. Initial state is enabled
	if !host.Registry.IsPluginEnabled(pluginID) {
		t.Errorf("expected plugin to be enabled initially")
	}

	st, ok := host.Registry.GetPluginStatus(pluginID)
	if !ok {
		t.Fatalf("expected status to exist for %s", pluginID)
	}
	if !st.Enabled {
		t.Errorf("status.Enabled = false, want true")
	}
	if st.IsActive {
		t.Errorf("status.IsActive = true, want false")
	}

	if _, found := host.Registry.FindPluginByCapability("tools"); !found {
		t.Errorf("expected capability text-processing to be found")
	}

	// 2. Disable plugin
	if err := setEnabled(t.Context(), host, pluginID, false); err != nil {
		t.Fatalf("switching the plugin failed: %v", err)
	}

	if host.Registry.IsPluginEnabled(pluginID) {
		t.Errorf("expected plugin to be disabled")
	}

	st, ok = host.Registry.GetPluginStatus(pluginID)
	if !ok || st.Enabled {
		t.Errorf("expected status.Enabled = false after disable")
	}

	if _, found := host.Registry.FindPluginByCapability("tools"); found {
		t.Errorf("expected disabled plugin to not be found by capability")
	}

	ctx := t.Context()
	triggered, err := host.Events.TriggerEvent(ctx, "on-edit")
	if err != nil {
		t.Fatalf("TriggerEvent error: %v", err)
	}
	if len(triggered) != 0 {
		t.Errorf("expected 0 triggered plugins for disabled plugin, got %d", len(triggered))
	}

	_, err = host.Activation.EnsureActive(ctx, pluginID)
	if err == nil {
		t.Errorf("expected EnsureActive to fail for disabled plugin")
	}

	// 3. Re-enable plugin
	if err := setEnabled(t.Context(), host, pluginID, true); err != nil {
		t.Fatalf("switching the plugin failed: %v", err)
	}

	if !host.Registry.IsPluginEnabled(pluginID) {
		t.Errorf("expected plugin to be enabled after toggle")
	}
	if _, found := host.Registry.FindPluginByCapability("tools"); !found {
		t.Errorf("expected capability text-processing to be found after re-enabling")
	}

	all := host.Registry.GetPluginStatuses()
	if len(all) != 1 || !all[0].Enabled {
		t.Errorf("expected 1 enabled plugin in GetPluginStatuses")
	}
}
