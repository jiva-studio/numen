package settings

import (
	"fmt"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/port/settingsport"
)

// CheckSettings validates settings against the declarations of the settings they are written to,
// and against the plugin that owns a plugin setting.
type CheckSettings struct {
	codec     settingsport.ValueCodec
	validator settingsport.WriteValidator
	plugins   settingsport.PluginSettings
}

// NewCheckSettings makes the use case. plugins is nil where no plugin host runs.
func NewCheckSettings(
	codec settingsport.ValueCodec, validator settingsport.WriteValidator, plugins settingsport.PluginSettings,
) *CheckSettings {
	return &CheckSettings{codec: codec, validator: validator, plugins: plugins}
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
	if err := u.validator.CheckWrite(one.Path, value); err != nil {
		return err
	}
	if u.plugins != nil && settingsdomain.IsPluginSetting(one.Path) {
		return u.plugins.CheckPluginSetting(one.Path[0], one.Path[1], value)
	}
	return nil
}
