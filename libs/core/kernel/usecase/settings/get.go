package settings

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// Current is the settings file as JSON and at its current position, with the plugins and the app
// settings that go with the settings.
type Current struct {
	Written string
	Version uint64
	Path    string
	Plugins []plugin.Info
	App     []plugin.Setting
	// AppGroups are the groups the settings of the app name, in the order their islands are drawn.
	AppGroups []manifest.SettingGroup
}

// Get reads every setting, the plugins and the settings the app declares.
type Get struct {
	read    func() (content, path string, version uint64, err error)
	plugins func() []plugin.Info
	app     func() []plugin.Setting
	groups  func() []manifest.SettingGroup
}

func NewGet(
	read func() (content, path string, version uint64, err error),
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
	content, path, version, err := u.read()
	if err != nil {
		return Current{}, err
	}
	pluginInfos := u.plugins()
	app := u.app()
	return Current{
		Written:   content,
		Version:   version,
		Path:      path,
		Plugins:   pluginInfos,
		App:       app,
		AppGroups: u.groups(),
	}, nil
}
