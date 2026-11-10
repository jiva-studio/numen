package settings

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
)

// Current is the settings file as JSON and at its current position, with the models and
// plugins that go with the settings.
type Current struct {
	Written string
	Path    string
	Models  []settingsdomain.Model
	Plugins []plugin.Info
	App     []plugin.Setting
}

// Get reads every setting, the models a setting can name, and the limits the
type Get struct {
	read    func() (content, path string, err error)
	models  func() []settingsdomain.Model
	plugins func() []plugin.Info
	app     func() []plugin.Setting
}

func NewGet(
	read func() (content, path string, err error),
	models func() []settingsdomain.Model,
	plugins func() []plugin.Info,
	app func() []plugin.Setting,
) *Get {
	return &Get{
		read:    read,
		models:  models,
		plugins: plugins,
		app:     app,
	}
}

func (u *Get) Execute() (Current, error) {
	content, path, err := u.read()
	if err != nil {
		return Current{}, err
	}
	var models []settingsdomain.Model
	if u.models != nil {
		models = u.models()
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
		Models:  models,
		Plugins: pluginInfos,
		App:     app,
	}, nil
}
