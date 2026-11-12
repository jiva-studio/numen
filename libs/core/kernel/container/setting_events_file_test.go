package container_test

import (
	"os"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
)

func TestTheLockFileNextToTheSettingsFileIsNoEvent(t *testing.T) {
	cfg, path, _ := openInstallation(t, `{}`)
	reader := listen(t, cfg)

	if err := os.WriteFile(path+".lock", nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path + ".lock"); err != nil {
		t.Fatal(err)
	}

	reader.expectNothing(t)
}

func TestSeveralWritesOfTheFileInQuickSuccessionArriveAsOneBatch(t *testing.T) {
	cfg, path, _ := openInstallation(t, `{"app":{"appearance":{"textScale":1}}}`)
	reader := listen(t, cfg)

	for _, scale := range []string{"1.1", "1.2", "1.3"} {
		body := `{"app":{"appearance":{"textScale":` + scale + `}}}`
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	expectChange(t, reader.await(t), []string{"app", "appearance", "textScale"}, `1.3`)
	reader.expectNothing(t)
}

func TestTheFileIsWatchedAgainWhenAReaderSubscribesAfterTheLastOneLeft(t *testing.T) {
	cfg, path, _ := openInstallation(t, `{"app":{"appearance":{"mode":"light"}}}`)
	_, stop := cfg.SubscribeSettingChanges()
	stop()
	reader := listen(t, cfg)

	if err := os.WriteFile(path, []byte(`{"app":{"appearance":{"mode":"dark"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	expectChange(t, reader.await(t), []string{"app", "appearance", "mode"}, `"dark"`)
}

func TestAWriteThatDisablesAPluginReachesThePluginHost(t *testing.T) {
	cfg, _, host := openInstallation(t, `{}`)
	reader := listen(t, cfg)

	if err := container.GetSettingsWriter(cfg)([]settings.Setting{
		{Path: []string{eventPlugin, "enabled"}, JSON: `false`},
	}); err != nil {
		t.Fatal(err)
	}

	expectChange(t, reader.await(t), []string{eventPlugin, "enabled"}, `false`)
	waitFor(t, func() bool { return len(host.listUpdates()) > 0 })
	if got := host.listUpdates(); !slices.Equal(got, []string{eventPlugin + ".enabled"}) {
		t.Errorf("the plugin host received %v", got)
	}
}

func TestAPluginHostReceivesNoChangeAfterItIsUnsubscribed(t *testing.T) {
	cfg, path, host := openInstallation(t, `{}`)
	container.UnsubscribeHost(cfg)

	if err := os.WriteFile(path, []byte(`{"`+eventPlugin+`":{"enabled":false}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	time.Sleep(quiet)

	if got := host.listUpdates(); len(got) != 0 {
		t.Errorf("the plugin host received %v after it was unsubscribed", got)
	}
}
