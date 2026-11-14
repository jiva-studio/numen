package settings

import (
	_ "embed"
	"encoding/json"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

// appDeclaration is the settings of the owner app in the format of the settings of a manifest.
//
//go:embed app.json
var appDeclaration []byte

// The keys of the declaration that code reads by name. The declaration holds each one's type,
// bounds and default.
const (
	keyTheme          = "appearance.theme"
	keyMode           = "appearance.mode"
	keyInterfaceScale = "appearance.interfaceScale"
	keyTextScale      = "appearance.textScale"
	keySyncNaming     = "naming.syncTitleAndFilename"
	keyProfile        = "performance.profile"
)

func listDetectedConnectionIDs(detected connectionport.Detected) []string {
	if detected == nil {
		return nil
	}
	var ids []string
	for _, one := range detected.ListDetected() {
		ids = append(ids, one.ID)
	}
	return ids
}

func listPendingDetections(types connectionport.Types, detected connectionport.Detected) []string {
	if types == nil || detected == nil {
		return nil
	}
	var ids []string
	for _, one := range types.ListTypes() {
		if detected.IsPending(one.GetID()) {
			ids = append(ids, one.GetID())
		}
	}
	return ids
}

// AddDetectedRegistries answers settingsdomain.RegistryDetectedConnections and settingsdomain.RegistryPendingDetections from the detection, and
// every other name from registries. A nil detection lists none.
func AddDetectedRegistries(registries settingsdomain.Registries, types connectionport.Types, detected connectionport.Detected) settingsdomain.Registries {
	return func(name string) ([]string, bool) {
		switch name {
		case settingsdomain.RegistryDetectedConnections:
			return listDetectedConnectionIDs(detected), true
		case settingsdomain.RegistryPendingDetections:
			return listPendingDetections(types, detected), true
		}
		return registries(name)
	}
}

// AddDetectedConnectionIDs adds to the values a window follows the ids of the connections the providers detected,
// under settingsdomain.DetectedConnectionsKey, so that the window reads the connections again when they change.
func AddDetectedConnectionIDs(values settingsdomain.Values, detected connectionport.Detected) {
	ids := listDetectedConnectionIDs(detected)
	if values == nil || len(ids) == 0 {
		return
	}
	if encoded, err := json.Marshal(ids); err == nil {
		values[settingsdomain.JoinPath([]string{settingsdomain.DetectedConnectionsKey})] = string(encoded)
	}
}

func readAppDeclaration() []manifest.Setting {
	decoded, err := manifestjson.Decode(appDeclaration)
	if err != nil {
		return nil
	}
	return decoded.Settings
}

// GetAppDeclaration returns the settings of the owner app, declared in the format of a manifest
// setting. A key is the path under the owner, joined by dots.
func GetAppDeclaration() []manifest.Setting {
	return readAppDeclaration()
}

func findAppSetting(key string) manifest.Setting {
	for _, one := range readAppDeclaration() {
		if one.Key == key {
			return one
		}
	}
	return manifest.Setting{}
}

func getAppDefault(key string) any {
	return findAppSetting(key).Default
}

func getAppString(key string) string {
	if text, isText := getAppDefault(key).(string); isText {
		return text
	}
	return ""
}

func getAppNumber(key string) float64 {
	if number, isNumber := getAppDefault(key).(float64); isNumber {
		return number
	}
	return 0
}

func getAppBool(key string) bool {
	if isOn, isBool := getAppDefault(key).(bool); isBool {
		return isOn
	}
	return false
}

func getAppBounds(key string) settingsdomain.Bounds {
	declared := findAppSetting(key)
	var bounds settingsdomain.Bounds
	if declared.Min != nil {
		bounds.Min = *declared.Min
	}
	if declared.Max != nil {
		bounds.Max = *declared.Max
	}
	return bounds
}

// ThemeRegistry answers the registry of themes with the themes list returns, and every other registry
// as unknown.
func ThemeRegistry(list func() []themedomain.Theme) RegistryOptions {
	return func(name string) (manifest.SettingOptions, bool) {
		if name != settingsdomain.RegistryThemes {
			return nil, false
		}
		var options manifest.SettingOptions
		for _, one := range list() {
			options = append(options, manifest.SettingOption{Value: one.Name, Title: one.Title})
		}
		return options, true
	}
}

// GetNames returns the registries by the values their options hold.
func (r RegistryOptions) GetNames() settingsdomain.Registries {
	return func(name string) ([]string, bool) {
		options, found := r(name)
		names := make([]string, 0, len(options))
		for _, one := range options {
			names = append(names, one.Value)
		}
		return names, found
	}
}
