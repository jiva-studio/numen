//go:build linux

package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
)

func runWithSettings(t *testing.T, content string, external *string) (string, bool) {
	t.Helper()
	config := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", config)
	if content != "" {
		dir := filepath.Join(config, "numen")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "numen.json"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv(dmabufEnv, "")
	if external == nil {
		if err := os.Unsetenv(dmabufEnv); err != nil {
			t.Fatal(err)
		}
	} else {
		t.Setenv(dmabufEnv, *external)
	}
	applyWebviewEnvironment(container.Config{})
	return os.LookupEnv(dmabufEnv)
}

func TestTheDmabufRendererIsDisabledWhenNothingIsSet(t *testing.T) {
	got, isSet := runWithSettings(t, "", nil)
	if !isSet || got != "1" {
		t.Errorf("the variable is %q (set: %v), want 1", got, isSet)
	}
}

func TestDmabufDisabledWhenTheFileNamesOtherSettingsOnly(t *testing.T) {
	got, isSet := runWithSettings(t, `{"v":1,"app":{"performance":{"profile":"high"}}}`, nil)
	if !isSet || got != "1" {
		t.Errorf("the variable is %q (set: %v), want 1", got, isSet)
	}
}

func TestTheDmabufRendererIsLeftAloneWhenTheFileTurnsTheSettingOff(t *testing.T) {
	if _, isSet := runWithSettings(t, `{"v":1,"app":{"performance":{"disableDmabufRenderer":false}}}`, nil); isSet {
		t.Error("the variable is set though the setting is off")
	}
}

func TestTheDmabufRendererIsDisabledWhenTheFileIsGarbage(t *testing.T) {
	got, isSet := runWithSettings(t, `{"app": [`, nil)
	if !isSet || got != "1" {
		t.Errorf("the variable is %q (set: %v), want 1", got, isSet)
	}
}

func TestAVariableTheUserSetWinsOverTheSetting(t *testing.T) {
	external := "0"
	got, _ := runWithSettings(t, `{"v":1,"app":{"performance":{"disableDmabufRenderer":true}}}`, &external)
	if got != "0" {
		t.Errorf("the variable is %q, want the user's 0", got)
	}
}

func TestReadingTheSettingWritesNoFile(t *testing.T) {
	runWithSettings(t, "", nil)
	if _, err := os.Stat(filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "numen", "numen.json")); err == nil {
		t.Error("the early read created the settings file")
	}
}

func runWithRegistry(t *testing.T, content string) (string, bool) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(dmabufEnv, "")
	if err := os.Unsetenv(dmabufEnv); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if content != "" {
		if err := os.WriteFile(filepath.Join(dir, "numen.json"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	applyWebviewEnvironment(container.Config{Paths: container.Paths{RegistryPath: filepath.Join(dir, "registry.json")}})
	return os.LookupEnv(dmabufEnv)
}

func TestTheSettingBesideTheRegistryTurnsTheDisablingOff(t *testing.T) {
	if _, isSet := runWithRegistry(t, `{"v":1,"app":{"performance":{"disableDmabufRenderer":false}}}`); isSet {
		t.Error("the variable is set though the file beside the registry turns the setting off")
	}
}

func TestTheSettingBesideTheRegistryTurnsTheDisablingOn(t *testing.T) {
	got, isSet := runWithRegistry(t, `{"v":1,"app":{"performance":{"disableDmabufRenderer":true}}}`)
	if !isSet || got != "1" {
		t.Errorf("the variable is %q (set: %v), want 1", got, isSet)
	}
}

func TestNoFileBesideTheRegistryDisablesTheRenderer(t *testing.T) {
	got, isSet := runWithRegistry(t, "")
	if !isSet || got != "1" {
		t.Errorf("the variable is %q (set: %v), want 1", got, isSet)
	}
}
