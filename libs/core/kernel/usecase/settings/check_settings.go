package settings

import (
	"fmt"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
)

// CheckSettings validates settings against the declarations of the settings they are written to,
// and against the plugin that owns a plugin setting.
type CheckSettings struct {
	codec      ValueCodec
	declared   settingsdomain.Declarations
	registries settingsdomain.Registries
	plugins    PluginSettings
}

// NewCheckSettings makes the use case. registries lists the values of the settings that draw their
// options from a registry. plugins is nil where no plugin host runs.
func NewCheckSettings(
	codec ValueCodec, declared settingsdomain.Declarations, registries settingsdomain.Registries,
	plugins PluginSettings,
) *CheckSettings {
	return &CheckSettings{codec: codec, declared: declared, registries: registries, plugins: plugins}
}

// Execute returns an error wrapping ErrNotASetting for the first entry that is refused.
func (u *CheckSettings) Execute(entries []settingsdomain.Setting) error {
	for _, one := range entries {
		if err := u.check(one); err != nil {
			return fmt.Errorf("%w: %w", settingsdomain.ErrNotASetting, err)
		}
	}
	return nil
}

func (u *CheckSettings) check(one settingsdomain.Setting) error {
	value, err := u.codec.Decode(one.JSON)
	if err != nil {
		return err
	}
	if err := settingsdomain.CheckValue(u.declared, u.registries, one.Path, value); err != nil {
		return err
	}
	if settingsdomain.IsPluginSetting(one.Path) {
		return u.plugins.CheckPluginSetting(one.Path[0], one.Path[1], value)
	}
	return nil
}
