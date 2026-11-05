package settings

import (
	_ "embed"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifestjson"
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

// RegistryThemes names the registry that lists the installed themes.
const RegistryThemes = "themes"

// readAppDeclaration decodes the embedded declaration.
func readAppDeclaration() []pluginport.PluginManifestSetting {
	decoded, err := manifestjson.Decode(appDeclaration)
	if err != nil {
		return nil
	}
	return decoded.Settings
}

// GetAppDeclaration returns the settings of the owner app, declared in the format of a manifest
// setting. A key is the path under the owner, joined by dots.
func GetAppDeclaration() []pluginport.PluginManifestSetting {
	return readAppDeclaration()
}

// findAppSetting returns the declaration of the setting of the owner app at a key.
func findAppSetting(key string) pluginport.PluginManifestSetting {
	for _, one := range readAppDeclaration() {
		if one.Key == key {
			return one
		}
	}
	return pluginport.PluginManifestSetting{}
}

// getAppDefault returns the default the declaration gives a key.
func getAppDefault(key string) any {
	return findAppSetting(key).Default
}

// getAppString returns the default of a string setting.
func getAppString(key string) string {
	if text, isText := getAppDefault(key).(string); isText {
		return text
	}
	return ""
}

// getAppNumber returns the default of a number setting.
func getAppNumber(key string) float64 {
	if number, isNumber := getAppDefault(key).(float64); isNumber {
		return number
	}
	return 0
}

// getAppBool returns the default of a boolean setting.
func getAppBool(key string) bool {
	if isOn, isBool := getAppDefault(key).(bool); isBool {
		return isOn
	}
	return false
}

// getAppBounds returns the bounds the declaration gives a range setting.
func getAppBounds(key string) Bounds {
	declared := findAppSetting(key)
	var bounds Bounds
	if declared.Min != nil {
		bounds.Min = *declared.Min
	}
	if declared.Max != nil {
		bounds.Max = *declared.Max
	}
	return bounds
}
