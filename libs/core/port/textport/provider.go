package textport

import "errors"

var (
	// ErrNoText is a file, a version or a key nothing has text for, or a
	// version whose file changed since it was recorded.
	ErrNoText = errors.New("there is no such text")
	// ErrNotTextPlugin is a plugin that is not enabled.
	ErrNotTextPlugin = errors.New("the plugin makes no text")
	// ErrInvalidVersion is a version with no key, no kind or no file, a file
	// that is not there or is not JSON Lines of segments, or a source that is
	// no other version of the file.
	ErrInvalidVersion = errors.New("the version is not valid")
	// ErrSourceNotAccepted is a source the plugin does not take.
	ErrSourceNotAccepted = errors.New("the plugin does not take this source")
)

// TextPlugins is the enabled plugins, which may record text versions.
type TextPlugins interface {
	// IsEnabled reports whether the plugin of the id is enabled.
	IsEnabled(pluginID string) bool
}
