package container

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
)

func writeAt(cfg Config, json string, path ...string) error {
	return getSettingsWriter(settingsOf(cfg))([]settings.Setting{{Path: path, JSON: json}})
}

func TestAProfileThatIsNoOptionIsRefusedAndNotSent(t *testing.T) {
	cfg, path := writeSettings(t, `{}`)
	host := &profileHost{}
	cfg.PluginHost = &PluginHost{Configuration: PluginConfiguration{SettingsPush: host}}

	err := writeAt(cfg, `"bogus"`, "app", "performance", "profile")
	if !errors.Is(err, settings.ErrNotASetting) {
		t.Fatalf("the write gave %v", err)
	}
	if len(host.profiles) != 0 {
		t.Errorf("profiles sent to the host = %v, want none", host.profiles)
	}
	raw, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if strings.Contains(string(raw), "bogus") {
		t.Errorf("the file holds the refused value:\n%s", raw)
	}
}

func TestAScaleOutsideItsBoundsIsRefusedWhoeverWritesIt(t *testing.T) {
	for _, value := range []string{`2.5`, `0.5`, `0`} {
		cfg, path := writeSettings(t, `{}`)
		err := writeAt(cfg, value, "app", "appearance", "interfaceScale")
		if !errors.Is(err, settings.ErrNotASetting) {
			t.Errorf("%s gave %v", value, err)
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if string(raw) != `{}` {
			t.Errorf("%s left the file as:\n%s", value, raw)
		}
	}
}

func TestAStringWhereTheNamingAcceptsABooleanIsRefused(t *testing.T) {
	cfg, _ := writeSettings(t, `{}`)
	err := writeAt(cfg, `"yes"`, "app", "naming", "syncTitleAndFilename")
	if !errors.Is(err, settings.ErrNotASetting) {
		t.Errorf("the write gave %v", err)
	}
}

func TestAThemeThatIsNotInstalledIsRefused(t *testing.T) {
	cfg, _ := writeSettings(t, `{}`)
	err := writeAt(cfg, `"mine:not-installed"`, "app", "appearance", "theme")
	if !errors.Is(err, settings.ErrNotASetting) {
		t.Errorf("the write gave %v", err)
	}
}

func TestAKeyAPluginDoesNotDeclareIsRefused(t *testing.T) {
	cfg, path := writeSettings(t, `{}`)
	err := writeAt(cfg, `"x"`, "md.numen.nobody", "mistyped")
	if !errors.Is(err, settings.ErrNotASetting) {
		t.Errorf("the write gave %v", err)
	}
	raw, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(raw) != `{}` {
		t.Errorf("the file is now:\n%s", raw)
	}
}

func TestAWriteOfSeveralSettingsIsRefusedWholeWhenOneIsRefused(t *testing.T) {
	cfg, path := writeSettings(t, `{}`)
	err := getSettingsWriter(settingsOf(cfg))([]settings.Setting{
		{Path: []string{"app", "appearance", "mode"}, JSON: `"dark"`},
		{Path: []string{"app", "appearance", "textScale"}, JSON: `9`},
	})
	if !errors.Is(err, settings.ErrNotASetting) {
		t.Fatalf("the write gave %v", err)
	}
	raw, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if strings.Contains(string(raw), "dark") {
		t.Errorf("the accepted half was written:\n%s", raw)
	}
}

const relayPlugin = "org.example.relay"

func writeRelayPlugin(t *testing.T, defaultPort int) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "relay")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := fmt.Sprintf(`{
		"id": %q,
		"name": "Relay",
		"version": "1.0.0",
		"targets": ["desktop.editor"],
		"backend": {"entrypoint": "bin/relay"},
		"contributes": {"listeners": [{"name": "relay", "portSetting": "port", "protocol": "tcp"}]},
		"settings": [
			{"key": "port", "type": "number", "title": "Port", "default": %d, "step": 1},
			{"key": "greeting", "type": "string", "title": "Greeting", "default": "good morning"}
		]
	}`, relayPlugin, defaultPort)
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func listenOnLoopback(t *testing.T) (net.Listener, int) {
	t.Helper()
	ln, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := ln.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("close the listener: %v", err)
		}
	})
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener address %v is not TCP", ln.Addr())
	}
	return ln, addr.Port
}

func TestSettingsWriteOfUsedPluginPortFailsAndKeepsTheFile(t *testing.T) {
	free, defaultPort := listenOnLoopback(t)
	if err := free.Close(); err != nil {
		t.Fatal(err)
	}
	_, takenPort := listenOnLoopback(t)
	cfg, path := writeSettings(t, `{}`)
	host := newPluginHost(t, "desktop.editor", writeRelayPlugin(t, defaultPort))
	t.Cleanup(func() {
		if err := host.Watch.TeardownAll(context.WithoutCancel(t.Context()), time.Millisecond); err != nil {
			t.Errorf("TeardownAll: %v", err)
		}
	})
	cfg.PluginHost = host

	err := writeAt(cfg, fmt.Sprint(takenPort), relayPlugin, "port")
	if !errors.Is(err, settings.ErrNotASetting) {
		t.Errorf("writing the taken port %d gave %v", takenPort, err)
	}
	raw, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(raw) != `{}` {
		t.Errorf("the file is now:\n%s", raw)
	}
	if err := writeAt(cfg, `"good evening"`, relayPlugin, "greeting"); err != nil {
		t.Errorf("writing greeting gave %v", err)
	}
}
