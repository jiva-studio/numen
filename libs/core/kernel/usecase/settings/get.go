package settings

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// Current is the settings file as JSON and at its current position, with the plugins and the app
// settings that go with the settings.
type Current struct {
	Written   string
	Version   uint64
	Path      string
	Plugins   []plugin.Info
	App       []plugin.Setting
	AppGroups []manifest.SettingGroup
}

// Get reads every setting, the plugins and the settings the app declares.
type Get struct {
	read            func() (settingsdomain.Snapshot, error)
	listPlugins     func() []plugin.Info
	listAppSettings func() []plugin.Setting
	listGroups      func() []manifest.SettingGroup
}

func NewGet(
	read func() (settingsdomain.Snapshot, error),
	listPlugins func() []plugin.Info,
	listAppSettings func() []plugin.Setting,
	listGroups func() []manifest.SettingGroup,
) *Get {
	if listPlugins == nil {
		listPlugins = func() []plugin.Info { return nil }
	}
	if listAppSettings == nil {
		listAppSettings = func() []plugin.Setting { return nil }
	}
	if listGroups == nil {
		listGroups = func() []manifest.SettingGroup { return nil }
	}
	return &Get{
		read:            read,
		listPlugins:     listPlugins,
		listAppSettings: listAppSettings,
		listGroups:      listGroups,
	}
}

func (u *Get) Execute() (Current, error) {
	snapshot, err := u.read()
	if err != nil {
		return Current{}, err
	}
	pluginInfos := u.listPlugins()
	app := u.listAppSettings()
	return Current{
		Written:   snapshot.Content,
		Version:   snapshot.Version,
		Path:      snapshot.Path,
		Plugins:   pluginInfos,
		App:       app,
		AppGroups: u.listGroups(),
	}, nil
}
