package file

import (
	"errors"
	"fmt"
	"strings"
)

// pluginScheme starts the address of a file in a plugin's store of the vault.
const pluginScheme = "plugin://"

// ErrInvalidAddress is an address that names no plugin or no file.
var ErrInvalidAddress = errors.New("the address names no file")

// Address is where a file is read from: Path in the store of the plugin PluginID, or Path in the
// vault where PluginID is empty.
type Address struct {
	PluginID string
	Path     string
}

// ParseAddress reads plugin://<plugin id>/<path> as a file in that plugin's store, and any other
// text as a path of the vault.
func ParseAddress(text string) (Address, error) {
	rest, isPluginFile := strings.CutPrefix(text, pluginScheme)
	if !isPluginFile {
		return Address{Path: text}, nil
	}
	pluginID, path, _ := strings.Cut(rest, "/")
	if pluginID == "" || path == "" {
		return Address{}, fmt.Errorf("%w: %q", ErrInvalidAddress, text)
	}
	return Address{PluginID: pluginID, Path: path}, nil
}

// IsPluginFile reports whether the address names a file in a plugin's store.
func (a Address) IsPluginFile() bool { return a.PluginID != "" }
