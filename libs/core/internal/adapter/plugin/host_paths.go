package plugin

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// The reasons a setting of type file or folder does not take a path.
var (
	errPathNotFound = errors.New("not found")
	errNotAFile     = errors.New("not a file")
	errNotAFolder   = errors.New("not a folder")
	errNotAccepted  = errors.New("wrong type of file")
)

// CheckPluginSetting reports why a setting the plugin declares does not take the value. A setting
// of type file takes an empty value or a file that exists with an extension it accepts; one of type
// folder an empty value or a folder that exists. Any other setting takes any value.
func (h *Host) CheckPluginSetting(pluginID, key string, value any) error {
	h.mu.RLock()
	p, ok := h.plugins[pluginID]
	h.mu.RUnlock()
	if !ok {
		return nil
	}
	declared, ok := p.Manifest.GetSetting(key)
	if !ok || (declared.Type != pluginport.SettingTypeFile && declared.Type != pluginport.SettingTypeFolder) {
		return nil
	}
	path, isText := value.(string)
	if !isText {
		return fmt.Errorf("%s: %w", key, errPathNotFound)
	}
	if path == "" {
		return nil
	}
	return checkPath(declared, path)
}

func checkPath(declared pluginport.PluginManifestSetting, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%s: %w", path, errPathNotFound)
	}
	if declared.Type == pluginport.SettingTypeFolder {
		if !info.IsDir() {
			return fmt.Errorf("%s: %w", path, errNotAFolder)
		}
		return nil
	}
	if info.IsDir() {
		return fmt.Errorf("%s: %w", path, errNotAFile)
	}
	extension := strings.ToLower(filepath.Ext(path))
	if len(declared.Accept) > 0 && !slices.ContainsFunc(declared.Accept, func(one string) bool { return strings.ToLower(one) == extension }) {
		return fmt.Errorf("%s: %w, expected %s", path, errNotAccepted, strings.Join(declared.Accept, ", "))
	}
	return nil
}
