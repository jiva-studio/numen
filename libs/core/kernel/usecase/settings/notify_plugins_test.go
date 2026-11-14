package settings_test

import (
	"slices"
	"sync"
	"testing"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
)

func setting(json string, path ...string) settingsdomain.Setting {
	return settingsdomain.Setting{Path: path, JSON: json}
}

func newNotifier(host *pluginHost, signal func(), report func(error)) *settings.NotifyPlugins {
	return settings.NewNotifyPlugins(settings.NotifyPluginsDeps{
		Codec: jsonCodec{}, Apply: host.UpdatePluginSetting, Config: host,
		ResolveProfile: func(profile string) string { return "resolved-" + profile }, Signal: signal, Report: report,
	})
}

// A plugin that is enabled in the same batch as its settings receives the settings first, and
// the settings of the shell reach no plugin.
func TestNotifyPluginsUpdatesTheEnabledFlagLast(t *testing.T) {
	host := &pluginHost{}
	signals := 0

	newNotifier(host, func() { signals++ }, nil).Execute([]settingsdomain.Setting{
		setting(`true`, "org.example.scans", "enabled"),
		setting(`3`, "org.example.scans", "depth"),
		setting(`"dark"`, "app", "appearance", "mode"),
	})

	if want := []string{"org.example.scans.depth", "org.example.scans.enabled"}; !slices.Equal(host.updates, want) {
		t.Errorf("updates = %v, want %v", host.updates, want)
	}
	if signals != 1 {
		t.Errorf("signals = %d, want 1", signals)
	}
}

// A change of the performance profile reaches the host as the profile the machine resolves it to,

func TestNotifyPluginsFollowsTheProfile(t *testing.T) {
	host := &pluginHost{}
	var reported []error

	notify := newNotifier(host, nil, func(err error) { reported = append(reported, err) })
	notify.Execute([]settingsdomain.Setting{
		setting(`"fast"`, "app", "performance", "profile"),
	})

	if !slices.Equal(host.profiles, []string{"resolved-fast"}) {
		t.Errorf("profiles = %v", host.profiles)
	}
	if len(reported) != 0 {
		t.Errorf("reported = %v", reported)
	}
}

// One host is subscribed once, however often the settings are applied to it, and it is delivered
// the batches until it is unsubscribed.
func TestApplyToPluginsSubscribesTheHostOnce(t *testing.T) {
	host := &pluginHost{}
	hosts := &settings.HostSubscriptions{}
	var mu sync.Mutex
	subscriptions := 0
	stopCalls := make(chan struct{})
	feed := make(chan []settingsdomain.Setting, 1)
	subscribe := func() (<-chan []settingsdomain.Setting, func()) {
		mu.Lock()
		subscriptions++
		mu.Unlock()
		return feed, func() { close(feed); close(stopCalls) }
	}
	file := &memoryFile{}
	write := settings.NewWriteSettings(settings.NewCheckSettings(jsonCodec{}, refusingValidator{declared: [][]string{{"org.example.scans", "depth"}}}, host), file, refreshRecorder{new([]string)})
	apply := settings.NewApplyToPlugins(settings.ApplyToPluginsDeps{
		Codec: jsonCodec{}, Config: host, Write: write, Notify: newNotifier(host, nil, nil), Subscribe: subscribe, Hosts: hosts,
	})

	apply.Execute(map[string]map[string]any{"org.example.scans": {"depth": 2.0}}, "fast")
	apply.Execute(nil, "fast")
	feed <- []settingsdomain.Setting{setting(`4`, "org.example.scans", "depth")}
	hosts.Unsubscribe(host)

	<-stopCalls
	if subscriptions != 1 {
		t.Errorf("subscriptions = %d, want 1", subscriptions)
	}
	if !slices.Equal(host.updates, []string{"org.example.scans.depth"}) {
		t.Errorf("updates = %v: the batch is delivered before Unsubscribe returns", host.updates)
	}
	if host.settings["org.example.scans"]["depth"] != 2.0 || !slices.Equal(host.profiles, []string{"fast", "fast"}) {
		t.Errorf("settings = %v, profiles = %v", host.settings, host.profiles)
	}
}

// A setting a plugin changes itself is saved through the same write as every other setting.
func TestApplyToPluginsSavesWhatAPluginChanges(t *testing.T) {
	host := &pluginHost{}
	file := &memoryFile{}
	write := settings.NewWriteSettings(settings.NewCheckSettings(jsonCodec{}, refusingValidator{declared: [][]string{{"org.example.scans", "depth"}}}, host), file, refreshRecorder{new([]string)})
	apply := settings.NewApplyToPlugins(settings.ApplyToPluginsDeps{
		Codec: jsonCodec{}, Config: host, Write: write, Notify: newNotifier(host, nil, nil), Hosts: &settings.HostSubscriptions{},
	})
	apply.Execute(nil, "fast")

	err := host.persist("org.example.scans", "depth", 5)

	if err != nil || len(file.writes) != 1 || file.writes[0].JSON != "5" {
		t.Fatalf("err = %v, writes = %+v", err, file.writes)
	}
}
