package settings

import (
	"context"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
)

// ApplyToPlugins gives the plugin host the settings, lets it save the settings a plugin changes
// itself, and has it follow every later change.
type ApplyToPlugins struct {
	codec     ValueCodec
	config    PluginConfig
	write     settingsWriter
	notify    pluginNotifier
	subscribe ChangeSubscriber
	hosts     hostSubscriber
}

// ApplyToPluginsDeps is what ApplyToPlugins is made of. The host is identified by Config, which every
// capability of the host is built from.
// Subscribe is nil where the changes are not followed.
type ApplyToPluginsDeps struct {
	Codec     ValueCodec
	Config    PluginConfig
	Write     settingsWriter
	Notify    pluginNotifier
	Subscribe ChangeSubscriber
	Hosts     hostSubscriber
}

// NewApplyToPlugins makes the use case.
func NewApplyToPlugins(deps ApplyToPluginsDeps) *ApplyToPlugins {
	hosts := deps.Hosts
	if deps.Subscribe == nil {
		hosts = ignoredSubscriptions{}
	}
	return &ApplyToPlugins{
		codec: deps.Codec, config: deps.Config, write: deps.Write, notify: deps.Notify,
		subscribe: deps.Subscribe, hosts: hosts,
	}
}

// Execute applies the settings of the plugins and the performance limits to the host, and
// subscribes the host to the setting changes once.
func (u *ApplyToPlugins) Execute(plugins map[string]map[string]any, performance plugindomain.Performance) {
	u.config.SetPerformance(context.Background(), performance)
	if plugins != nil {
		u.config.SetSettings(plugins)
	}
	u.config.SetPersistHandler(u.persist)
	u.hosts.Subscribe(u.config, u.subscribe, u.notify.Execute)
}

func (u *ApplyToPlugins) persist(pluginID, key string, value any) error {
	raw, err := u.codec.Encode(value)
	if err != nil {
		return err
	}
	return u.write.Execute([]settingsdomain.Setting{{Path: []string{pluginID, key}, JSON: raw}})
}

// settingsWriter saves settings.
type settingsWriter interface {
	Execute(entries []settingsdomain.Setting) error
}

// pluginNotifier delivers a batch of setting changes to the plugins that watch them.
type pluginNotifier interface {
	Execute(batch []settingsdomain.Setting)
}

// hostSubscriber delivers the setting changes to a host once.
type hostSubscriber interface {
	Subscribe(host any, subscribe ChangeSubscriber, deliver func([]settingsdomain.Setting))
}

// ignoredSubscriptions follows no setting changes.
type ignoredSubscriptions struct{}

func (ignoredSubscriptions) Subscribe(any, ChangeSubscriber, func([]settingsdomain.Setting)) {
}
