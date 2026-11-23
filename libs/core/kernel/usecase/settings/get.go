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
	read    func() (settingsdomain.Snapshot, error)
	plugins func() []plugin.Info
	app     func() []plugin.Setting
	groups  func() []manifest.SettingGroup
}

func NewGet(
	read func() (settingsdomain.Snapshot, error),
	plugins func() []plugin.Info,
	app func() []plugin.Setting,
	groups func() []manifest.SettingGroup,
) *Get {
	if plugins == nil {
		plugins = func() []plugin.Info { return nil }
	}
	if app == nil {
		app = func() []plugin.Setting { return nil }
	}
	if groups == nil {
		groups = func() []manifest.SettingGroup { return nil }
	}
	return &Get{
		read:    read,
		plugins: plugins,
		app:     app,
		groups:  groups,
	}
}

func (u *Get) Execute() (Current, error) {
	snapshot, err := u.read()
	if err != nil {
		return Current{}, err
	}
	pluginInfos := u.plugins()
	app := u.app()
	return Current{
		Written:   snapshot.Content,
		Version:   snapshot.Version,
		Path:      snapshot.Path,
		Plugins:   pluginInfos,
		App:       app,
		AppGroups: u.groups(),
	}, nil
}
