package settings

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// Current is the settings file as JSON and at its current position, with the plugins and the app
// settings that go with the settings.
type Current struct {
	Written string
	Path    string
	Plugins []plugin.Info
	App     []plugin.Setting
}

// Get reads every setting, the plugins and the settings the app declares.
type Get struct {
	read    func() (content, path string, err error)
	plugins func() []plugin.Info
	app     func() []plugin.Setting
}

func NewGet(
	read func() (content, path string, err error),
	plugins func() []plugin.Info,
	app func() []plugin.Setting,
) *Get {
	return &Get{
		read:    read,
		plugins: plugins,
		app:     app,
	}
}

func (u *Get) Execute() (Current, error) {
	content, path, err := u.read()
	if err != nil {
		return Current{}, err
	}
	var pluginInfos []plugin.Info
	if u.plugins != nil {
		pluginInfos = u.plugins()
	}
	var app []plugin.Setting
	if u.app != nil {
		app = u.app()
	}
	return Current{
		Written: content,
		Path:    path,
		Plugins: pluginInfos,
		App:     app,
	}, nil
}
