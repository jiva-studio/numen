package settings

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// GetOwner returns the values stored under an owner, the plugin id or OwnerApp.
func (f *File) GetOwner(owner string) map[string]any {
	if owner != settingsdomain.AppOwner {
		return maps.Clone(f.Plugins[owner])
	}
	content, err := f.MarshalJSON()
	if err != nil {
		return nil
	}
	var root map[string]any
	if err := json.Unmarshal(content, &root); err != nil {
		return nil
	}
	if app, isObject := root[settingsdomain.AppOwner].(map[string]any); isObject {
		return app
	}
	return nil
}

// GetValue returns the value in force at a path: the stored value, else the default its owner
// declares. The first element of the path is the owner.
func (f *File) GetValue(path ...string) (any, bool) {
	if len(path) < 2 {
		return nil, false
	}
	if value, found := lookupPath(f.GetOwner(path[0]), path[1:]); found && value != nil {
		return value, true
	}
	if path[0] == settingsdomain.AppOwner {
		if value := getAppDefault(strings.Join(path[1:], ".")); value != nil {
			return value, true
		}
	}
	return nil, false
}

// DropInvalid leaves out of force every stored value that settingsdomain.CheckValue refuses for the declarations
// given: an undeclared key, a value of the wrong type or outside its bounds, a value that is no
// option, and an object of an owner that has no declaration. In a setting of type connections only
// the rows that fail are left out, and the rows beyond max, each with a notice. Each one adds a notice to the file
// and is applied nowhere. The bytes of the file on disk are not touched.
func (f *File) DropInvalid(declared settingsdomain.Declarations, registries settingsdomain.Registries) {
	registries = withConnectionIDs(f.Connections, registries)
	f.dropInvalidApp(declared, registries)
	for _, id := range slices.Sorted(maps.Keys(f.Plugins)) {
		settings, found := declared(id)
		if !found {
			delete(f.Plugins, id)
			f.AddNotice(fmt.Sprintf("numen.json: no installed plugin %s; its settings are ignored", id))
			continue
		}
		f.dropInvalidRows(id, settings, registries)
		for _, key := range slices.Sorted(maps.Keys(f.Plugins[id])) {
			if err := settingsdomain.CheckValue(declared, registries, []string{id, key}, f.Plugins[id][key]); err != nil {
				delete(f.Plugins[id], key)
				f.AddNotice(fmt.Sprintf("numen.json: %v; the value is ignored", err))
			}
		}
	}
}

func (f *File) dropInvalidApp(declared settingsdomain.Declarations, registries settingsdomain.Registries) {
	for _, one := range readAppDeclaration() {
		path := append([]string{settingsdomain.AppOwner}, strings.Split(one.Key, ".")...)
		value, found := f.GetValue(path...)
		if !found || value == one.Default {
			continue
		}
		if err := settingsdomain.CheckValue(declared, registries, path, value); err != nil {
			f.resetApp(one)
			f.AddNotice(fmt.Sprintf("numen.json: %v; the default %v applies", err, one.Default))
		}
	}
}

func (f *File) resetApp(declared manifest.Setting) {
	switch declared.Key {
	case keyTheme:
		f.Appearance.Theme = getAppString(keyTheme)
	case keyMode:
		f.Appearance.Mode = getAppString(keyMode)
	case keyInterfaceScale:
		f.Appearance.InterfaceScale = getAppNumber(keyInterfaceScale)
	case keyTextScale:
		f.Appearance.TextScale = getAppNumber(keyTextScale)
	case keySyncNaming:
		f.Titles = DefaultTitles()
	case keyProfile:
		f.Performance = DefaultPerformance()
	}
}

// RegistryOptions lists the options a registry offers to the settings that draw on it, and false for
// a registry that is not known.
type RegistryOptions func(name string) (manifest.SettingOptions, bool)

func (r RegistryOptions) ListRegistryOptions(name string) (manifest.SettingOptions, bool) {
	return r(name)
}
