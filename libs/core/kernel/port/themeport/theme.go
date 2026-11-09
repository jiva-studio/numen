package themeport

import "errors"

// ErrNoTheme is a theme no enabled plugin contributes.
var ErrNoTheme = errors.New("no such plugin theme")

// PluginTheme is a theme an enabled plugin contributes.
type PluginTheme struct {
	PluginID   string
	ID         string
	Name       string
	Appearance string
}

// PluginThemes lists and reads the themes of the enabled plugins of this installation's target. It
// answers from the discovered manifests and starts no plugin process.
type PluginThemes interface {
	// ListThemes returns the themes in the order of the plugin ids, and in the order a plugin declares
	// them.
	ListThemes() []PluginTheme
	// ReadTheme returns the stylesheet of the theme, or ErrNoTheme when no enabled plugin contributes it.
	ReadTheme(pluginID, themeID string) (string, error)
}
