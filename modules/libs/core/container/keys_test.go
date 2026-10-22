package container

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// keyJar is a key store held in memory.
type keyJar map[string]string

func (j keyJar) GetKey(_ context.Context, name string) (string, error) {
	if key, found := j[name]; found {
		return key, nil
	}
	return "", port.ErrNoKey
}

func (j keyJar) SetKey(_ context.Context, name, key string) error {
	j[name] = key
	return nil
}

func (j keyJar) DeleteKey(_ context.Context, name string) error {
	if _, found := j[name]; !found {
		return port.ErrNoKey
	}
	delete(j, name)
	return nil
}

// writeSecretPlugin puts a plugin that declares the apiKey secret in a folder
// and returns the folder.
func writeSecretPlugin(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "agent")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
		"id": "md.numen.agents.numen",
		"name": "Agent",
		"version": "1.0.0",
		"capabilities": ["ai.agent.provider"],
		"targets": ["desktop.editor"],
		"secrets": ["apiKey"],
		"backend": {"type": "process", "entrypoint": "bin/agent", "serviceName": "numen.v1.AgentService"}
	}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

// An installation names its key store, and otherwise has the system keychain.
func TestTheKeyStoreIsTheOneNamedElseTheSystemKeychain(t *testing.T) {
	jar := keyJar{}
	if got := (Config{KeyStore: jar}).GetKeyStore(); got == nil {
		t.Fatal("no key store")
	} else if err := got.SetKey(t.Context(), "n", "k"); err != nil || jar["n"] != "k" {
		t.Errorf("the named store was not used: %v, %v", jar, err)
	}
	if (Config{}).GetKeyStore() == nil {
		t.Error("an installation naming none has no key store")
	}
}

// A secret written through the container lands in the key store the plugin
// host reads its plugins' secrets from, under the plugin and the name, and the
// settings file records the name of the secret and never its key.
func TestASecretSavedThroughTheContainerReachesTheKeyStore(t *testing.T) {
	jar := keyJar{}
	settingsPath := filepath.Join(t.TempDir(), "numen.json")
	cfg := Config{KeyStore: jar, SettingsPath: settingsPath, PluginHost: NewPluginHost("desktop.editor", writeSecretPlugin(t))}
	cfg = cfg.SetSettings(DefaultSettings())

	if err := cfg.GetPluginSecretWriter()(t.Context(), "md.numen.agents.numen", "apiKey", "value-1"); err != nil {
		t.Fatal(err)
	}
	if jar["md.numen.agents.numen/apiKey"] != "value-1" {
		t.Errorf("the store holds %v", jar)
	}
	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), `"storedSecrets"`) || strings.Contains(string(content), "value-1") {
		t.Errorf("the settings file holds:\n%s", content)
	}
	if err := cfg.GetPluginSecretDeleter()(t.Context(), "md.numen.agents.numen", "apiKey"); err != nil {
		t.Fatal(err)
	}
	if len(jar) != 0 {
		t.Errorf("the store holds %v after a delete", jar)
	}
}

// A secret the plugin does not declare is refused.
func TestASecretNoPluginDeclaresIsRefused(t *testing.T) {
	jar := keyJar{}
	cfg := Config{KeyStore: jar, SettingsPath: filepath.Join(t.TempDir(), "numen.json"), PluginHost: NewPluginHost("desktop.editor", writeSecretPlugin(t))}
	cfg = cfg.SetSettings(DefaultSettings())
	if err := cfg.GetPluginSecretWriter()(t.Context(), "md.numen.agents.numen", "other", "value"); err == nil {
		t.Error("an undeclared secret was stored")
	}
	if len(jar) != 0 {
		t.Errorf("the store holds %v", jar)
	}
}

// An installation without a plugin host has no plugin to keep a secret for.
func TestWithoutAPluginHostASecretIsAnError(t *testing.T) {
	cfg := Config{}
	if err := cfg.GetPluginSecretWriter()(t.Context(), "p", "n", "k"); !errors.Is(err, errNoPluginHost) {
		t.Errorf("write: error = %v", err)
	}
	if err := cfg.GetPluginSecretDeleter()(t.Context(), "p", "n"); !errors.Is(err, errNoPluginHost) {
		t.Errorf("delete: error = %v", err)
	}
}
