package container

import (
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/commandbridge"
	"github.com/jiva-studio/numen/libs/core/usecase/settings"
	"github.com/jiva-studio/numen/libs/core/usecase/trigger"
)

// openTriggers is the triggers of the enabled plugins over the files of a vault, which read the
// settings as get reads them; nil where the installation runs no plugins.
func (c Config) openTriggers(get *settings.Get) *trigger.Triggers {
	if c.PluginHost == nil || c.SafeMode || get == nil {
		return nil
	}
	return trigger.NewTriggers(
		commandbridge.Open(c.PluginHost), c.VaultReaders(), c.PluginHost,
		derivedfile.TriggerRecords{Options: c.vaultOptions(), Folder: coreFolder},
		func() (map[string]any, error) {
			current, err := get.Execute()
			if err != nil {
				return nil, err
			}
			return trigger.CreateConfig(current.Written, current.App, current.Plugins)
		},
	)
}
