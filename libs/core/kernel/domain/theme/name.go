package theme

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Extension is what a file is called to be a theme.
const Extension = ".css"

// MaxSize is the largest file read as a theme. Its text is spliced into the
// page before the window draws, and the presets are a few kilobytes each.
const MaxSize = 256 << 10

// PluginPrefix starts the name of a plugin theme: `plugin:<pluginId>/<themeId>`.
const PluginPrefix = string(Plugin) + ":"

// PluginName is the name of the theme a plugin contributes.
func PluginName(pluginID, themeID string) string {
	return PluginPrefix + pluginID + "/" + themeID
}

// GetName is the name of the theme of the title on the origin.
func GetName(origin Origin, title string) string {
	return string(origin) + ":" + title
}

// SplitPluginName returns the plugin id and the theme id a plugin theme name holds.
func SplitPluginName(name string) (pluginID, themeID string, isValid bool) {
	pluginID, themeID, found := strings.Cut(strings.TrimPrefix(name, PluginPrefix), "/")
	return pluginID, themeID, found && pluginID != "" && themeID != ""
}

// SplitName returns the origin and the title of a preset or own theme name, or an error wrapping
// ErrNoTheme where the name is of neither or its title could leave the folder.
func SplitName(name string) (Origin, string, error) {
	origin, title, found := strings.Cut(name, ":")
	if found && IsPlainTitle(title) {
		switch Origin(origin) {
		case Preset, Mine:
			return Origin(origin), title, nil
		case Plugin:
			// A plugin theme is named by plugin and theme id and is resolved by its own source.
		}
	}
	return "", "", fmt.Errorf("%s: %w", name, ErrNoTheme)
}

// IsPlainTitle reports whether the title is one file name: it holds no separator and leads nowhere else.
func IsPlainTitle(title string) bool {
	if title == "" || title == "." || title == ".." {
		return false
	}
	if strings.ContainsAny(title, `/\`) || strings.ContainsRune(title, 0) {
		return false
	}
	return title == filepath.Base(title)
}
