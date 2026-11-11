package settings

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// ErrNoPathSetting is a plugin or a setting of it that holds no path.
var ErrNoPathSetting = errors.New("the plugin declares no setting of type file under that key")

type ChosenPath struct {
	Path     string
	IsChosen bool
}

// ChoosePluginPath opens the machine's native file picker for a setting of
// type file, and writes the chosen path.
type ChoosePluginPath struct {
	dialog  machineport.PathDialog
	plugins func() []plugindomain.Info
	write   func(settings []settingsdomain.Setting) error
}

// NewChoosePluginPath makes the use case over the picker and the plugins:
// write writes settings as Write does.
func NewChoosePluginPath(
	dialog machineport.PathDialog, plugins func() []plugindomain.Info, write func(settings []settingsdomain.Setting) error,
) *ChoosePluginPath {
	return &ChoosePluginPath{dialog: dialog, plugins: plugins, write: write}
}

// Execute prompts for the path and writes it. A path the setting does not accept
// returns an error wrapping ErrNotASetting, and nothing is written.
func (u *ChoosePluginPath) Execute(ctx context.Context, pluginID, key string) (ChosenPath, error) {
	declared, err := u.findSetting(pluginID, key)
	if err != nil {
		return ChosenPath{}, err
	}
	path, isChosen, err := u.dialog.ChooseFile(ctx, declared.Title, "", declared.Accept)
	if err != nil || !isChosen {
		return ChosenPath{}, err
	}
	result := ChosenPath{Path: path, IsChosen: true}
	return result, u.write([]settingsdomain.Setting{{Path: []string{pluginID, key}, JSON: strconv.Quote(path)}})
}

func (u *ChoosePluginPath) findSetting(pluginID, key string) (plugindomain.Setting, error) {
	for _, plugin := range u.plugins() {
		if plugin.ID != pluginID {
			continue
		}
		for _, declared := range plugin.Settings {
			if declared.Key == key && declared.Type == manifest.SettingTypeFile {
				return declared, nil
			}
		}
	}
	return plugindomain.Setting{}, fmt.Errorf("%w: %s %s", ErrNoPathSetting, pluginID, key)
}
