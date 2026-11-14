package settings

import (
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// dropInvalidRows leaves out of force, in each setting of rows of the owner, the rows that
// settingsdomain.FilterRows leaves out, with a notice each.
func (f *File) dropInvalidRows(owner string, settings []manifest.Setting, registries settingsdomain.Registries) {
	for _, declared := range settings {
		if declared.Type != manifest.SettingTypeConnections {
			continue
		}
		list, isList := f.Plugins[owner][declared.Key].([]any)
		if !isList {
			continue
		}
		remainder, issues := settingsdomain.FilterRows(declared, registries, list)
		for _, issue := range issues {
			f.AddNotice(fmt.Sprintf("numen.json: %s.%s row %d: %s; the row is ignored", owner, declared.Key, issue.Index, issue.Reason))
		}
		f.Plugins[owner][declared.Key] = remainder
	}
}

func withConnectionIDs(connections []connection.Connection, registries settingsdomain.Registries) settingsdomain.Registries {
	ids := make([]string, 0, len(connections))
	for _, one := range connections {
		ids = append(ids, one.ID)
	}
	return func(name string) ([]string, bool) {
		if name == settingsdomain.RegistryConnections {
			return ids, true
		}
		if registries == nil {
			return nil, false
		}
		return registries(name)
	}
}
