package settings

import (
	"context"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/port/settingsport"
)

// ApplyToPlugins gives the plugin host the settings, lets it save the settings a plugin changes
// itself, and has it follow every later change.
type ApplyToPlugins struct {
	codec     settingsport.ValueCodec
	config    settingsport.PluginConfig
	write     settingsWriting
	notify    pluginNotifying
	subscribe settingsport.ChangeSubscriber
	hosts     hostSubscribing
}

// ApplyToPluginsDeps is what ApplyToPlugins is made of. The host is identified by Config, which every
// capability of the host is built from.
// Subscribe is nil where the changes are not followed.
type ApplyToPluginsDeps struct {
	Codec     settingsport.ValueCodec
	Config    settingsport.PluginConfig
	Write     settingsWriting
	Notify    pluginNotifying
	Subscribe settingsport.ChangeSubscriber
	Hosts     hostSubscribing
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

// Execute applies the settings of the plugins and the performance profile to the host, and
// subscribes the host to the setting changes once.
func (u *ApplyToPlugins) Execute(plugins map[string]map[string]any, profile string) {
	u.config.SetPerformanceProfile(context.Background(), profile)
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

// settingsWriting saves settings.
type settingsWriting interface {
	Execute(entries []settingsdomain.Setting) error
}

// pluginNotifying delivers a batch of setting changes to the plugins that watch them.
type pluginNotifying interface {
	Execute(batch []settingsdomain.Setting)
}

// hostSubscribing delivers the setting changes to a host once.
type hostSubscribing interface {
	Subscribe(host any, subscribe settingsport.ChangeSubscriber, deliver func([]settingsdomain.Setting))
}

// ignoredSubscriptions follows no setting changes.
type ignoredSubscriptions struct{}

func (ignoredSubscriptions) Subscribe(any, settingsport.ChangeSubscriber, func([]settingsdomain.Setting)) {
}
