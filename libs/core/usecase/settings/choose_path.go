package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// ErrNoPathSetting is a plugin or a setting of it that holds no path.
var ErrNoPathSetting = errors.New("the plugin declares no setting of type file or folder under that key")

// ChosenPath is the path a person chose for a setting.
type ChosenPath struct {
	Path     string
	IsChosen bool
}

// ChoosePluginPath puts the machine's own file or folder picker in front of the person for a
// setting of type file or folder, and writes the path they chose.
type ChoosePluginPath struct {
	dialog  machineport.PathDialog
	plugins func() []PluginInfo
	write   func(settings []Setting) error
}

// NewChoosePluginPath makes the use case over the picker and the plugins:
// write writes settings as Write does.
func NewChoosePluginPath(
	dialog machineport.PathDialog, plugins func() []PluginInfo, write func(settings []Setting) error,
) *ChoosePluginPath {
	return &ChoosePluginPath{dialog: dialog, plugins: plugins, write: write}
}

// Execute asks the person for the path and writes it. A path the setting does not take answers an
// error wrapping ErrNotASetting, and nothing is written.
func (u *ChoosePluginPath) Execute(ctx context.Context, pluginID, key string) (ChosenPath, error) {
	declared, err := u.findSetting(pluginID, key)
	if err != nil {
		return ChosenPath{}, err
	}
	var path string
	var isChosen bool
	if declared.Type == pluginport.SettingTypeFolder {
		path, isChosen, err = u.dialog.ChooseFolder(ctx, declared.Title, "")
	} else {
		path, isChosen, err = u.dialog.ChooseFile(ctx, declared.Title, "", declared.Accept)
	}
	if err != nil || !isChosen {
		return ChosenPath{}, err
	}
	written, err := json.Marshal(path)
	if err != nil {
		return ChosenPath{}, err
	}
	result := ChosenPath{Path: path, IsChosen: true}
	return result, u.write([]Setting{{Path: []string{"plugins", pluginID, key}, JSON: string(written)}})
}

func (u *ChoosePluginPath) findSetting(pluginID, key string) (PluginSetting, error) {
	for _, plugin := range u.plugins() {
		if plugin.ID != pluginID {
			continue
		}
		for _, declared := range plugin.Settings {
			if declared.Key == key && (declared.Type == pluginport.SettingTypeFile || declared.Type == pluginport.SettingTypeFolder) {
				return declared, nil
			}
		}
	}
	return PluginSetting{}, fmt.Errorf("%w: %s %s", ErrNoPathSetting, pluginID, key)
}
