package settings_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const statusBarPlugin = "md.numen.views.statusBar"

func TestAnUntouchedInstallationDisablesTheStatusBarPlugin(t *testing.T) {
	path := filepath.Join(t.TempDir(), "numen.json")

	cfg, err := openAt(path)
	if err != nil {
		t.Fatal(err)
	}

	if enabled, found := cfg.Plugins[statusBarPlugin]["enabled"]; !found || enabled != false {
		t.Errorf("the status bar plugin is enabled = %v (set %v), want false", enabled, found)
	}
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(written), statusBarPlugin) {
		t.Errorf("the written file names no %s:\n%s", statusBarPlugin, written)
	}
}

func TestAFileWithoutTheStatusBarPluginKeepsItDisabled(t *testing.T) {
	cfg, err := openAt(write(t, `{"v": 1, "org.example.lantern": {"enabled": true}}`))
	if err != nil {
		t.Fatal(err)
	}

	if enabled := cfg.Plugins[statusBarPlugin]["enabled"]; enabled != false {
		t.Errorf("the status bar plugin is enabled = %v, want false", enabled)
	}
}

func TestAFileThatEnablesTheStatusBarPluginKeepsItEnabled(t *testing.T) {
	cfg, err := openAt(write(t, `{"v": 1, "`+statusBarPlugin+`": {"enabled": true}}`))
	if err != nil {
		t.Fatal(err)
	}

	if enabled := cfg.Plugins[statusBarPlugin]["enabled"]; enabled != true {
		t.Errorf("the status bar plugin is enabled = %v, want true", enabled)
	}
}
