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
	write     *WriteSettings
	notify    *NotifyPlugins
	subscribe settingsport.ChangeSubscriber
	hosts     *HostSubscriptions
	setup     func()
}

// NewApplyToPlugins makes the use case. The host is identified by config, which every capability
// of the host is built from. setup hands the host what it needs besides settings, such as the key
// store. subscribe is nil where the changes are not followed.
func NewApplyToPlugins(
	codec settingsport.ValueCodec, config settingsport.PluginConfig, write *WriteSettings, notify *NotifyPlugins,
	subscribe settingsport.ChangeSubscriber, hosts *HostSubscriptions, setup func(),
) *ApplyToPlugins {
	return &ApplyToPlugins{codec: codec, config: config, write: write, notify: notify, subscribe: subscribe, hosts: hosts, setup: setup}
}

// Execute applies the settings of the plugins and the performance profile to the host, and
// subscribes the host to the setting changes once.
func (u *ApplyToPlugins) Execute(plugins map[string]map[string]any, profile string) {
	u.config.SetPerformanceProfile(context.Background(), profile)
	if plugins != nil {
		u.config.SetSettings(plugins)
	}
	u.config.SetPersistHandler(u.persist)
	u.setup()
	if u.subscribe != nil {
		u.hosts.Subscribe(u.config, u.subscribe, u.notify.Execute)
	}
}

func (u *ApplyToPlugins) persist(pluginID, key string, value any) error {
	raw, err := u.codec.Encode(value)
	if err != nil {
		return err
	}
	return u.write.Execute([]settingsdomain.Setting{{Path: []string{pluginID, key}, JSON: raw}})
}
