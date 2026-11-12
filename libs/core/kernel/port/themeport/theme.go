package themeport

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
)

// PluginThemes lists and reads the themes of the enabled plugins of this installation's target. It
// answers from the discovered manifests and starts no plugin process.
type PluginThemes interface {
	// ListThemes returns the themes in the order of the plugin ids, and in the order a plugin declares
	// them.
	ListThemes() []theme.PluginTheme
	// ReadTheme returns the stylesheet of the theme, or theme.ErrNoTheme when no enabled plugin contributes it.
	ReadTheme(pluginID, themeID string) (string, error)
}

// PresetThemes is the themes that ship inside the application.
type PresetThemes interface {
	// ListPresets returns the presets, each readable.
	ListPresets() []theme.File
	// ReadPreset returns the stylesheet of the preset, or an error where the title names none.
	ReadPreset(title string) (string, error)
}

// OwnThemes is the themes in the user's folder.
type OwnThemes interface {
	// ListOwnThemes returns the readable themes of the folder, none where the folder cannot be read.
	ListOwnThemes() []theme.File
	// ReadOwnTheme returns the stylesheet of the title, theme.ErrNoTheme where the folder has none,
	// or theme.ErrNoFolder where there is no folder.
	ReadOwnTheme(title string) (string, error)
}

// OwnThemeWatch reports changes to the user's folder.
type OwnThemeWatch interface {
	// WatchOwnThemes reports the titles of the themes that changed, batched over the hold, until ctx
	// ends, and closes the channel behind it. It returns theme.ErrNoFolder where there is no folder.
	WatchOwnThemes(ctx context.Context, hold time.Duration) (<-chan []string, error)
}

// AppearanceReader reads the stored appearance.
type AppearanceReader interface {
	// ReadAppearance returns the appearance the settings hold.
	ReadAppearance() (theme.Appearance, error)
}

// AppearanceWriter stores the appearance.
type AppearanceWriter interface {
	// WriteAppearance persists the appearance. A scale outside its bounds is refused without writing.
	WriteAppearance(theme.Appearance) error
}

// ThemeWarnings shows the user a configuration issue.
type ThemeWarnings interface {
	// WarnTheme reports the issue. An empty text withdraws the warning an earlier call reported.
	WarnTheme(text string)
}
