package container_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
)

// A plugin host receives the setting changes once, however often the settings are applied
// to it and from whichever copy of the configuration: a write reaches the plugin once.
func TestAPluginHostIsSubscribedOncePerHost(t *testing.T) {
	cfg, _, host := openInstallation(t, `{}`)
	cfg.SetSettings(container.DefaultSettings())
	container.NewConfig(cfg).SetSettings(container.DefaultSettings())
	reader := listen(t, cfg)

	err := container.GetSettingsWriter(cfg)([]settings.Setting{{Path: []string{eventPlugin, "mode"}, JSON: `"b"`}})
	if err != nil {
		t.Fatal(err)
	}
	reader.await(t)
	reader.expectNothing(t)
	for range 500 {
		if len(host.listUpdates()) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(quiet)

	if got := host.listUpdates(); len(got) != 1 || got[0] != eventPlugin+".mode" {
		t.Errorf("host updates = %v, want one update of %s.mode", got, eventPlugin)
	}
}
