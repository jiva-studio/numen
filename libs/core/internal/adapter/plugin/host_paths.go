package plugin

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

// The reasons a setting of type file does not take a path.
var (
	errPathNotFound = errors.New("not found")
	errNotAFile     = errors.New("not a file")
	errNotAccepted  = errors.New("wrong type of file")
)

// CheckPluginSetting reports why a setting the plugin declares does not take the value. A setting
// of type file takes an empty value or a file that exists with an extension it accepts. Any other
// setting takes any value.
func (h *Host) CheckPluginSetting(pluginID, key string, value any) error {
	h.mu.RLock()
	p, ok := h.plugins[pluginID]
	h.mu.RUnlock()
	if !ok {
		return nil
	}
	declared, ok := p.Manifest.GetSetting(key)
	if !ok || declared.Type != pluginport.SettingTypeFile {
		return nil
	}
	path, isText := value.(string)
	if !isText {
		return fmt.Errorf("%s: %w", key, errPathNotFound)
	}
	if path == "" {
		return nil
	}
	for one := range strings.SplitSeq(path, "\n") {
		entry := strings.TrimSpace(one)
		if entry == "" {
			continue
		}
		if err := checkPath(declared, entry); err != nil {
			return err
		}
	}
	return nil
}

// GetPluginSettingDeclarations returns the settings the manifest of the plugin declares, and false
// for a plugin the host does not know.
func (h *Host) GetPluginSettingDeclarations(pluginID string) ([]pluginport.PluginManifestSetting, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	p, ok := h.plugins[pluginID]
	if !ok {
		return nil, false
	}
	return slices.Clone(p.Manifest.Settings), true
}

func checkPath(declared pluginport.PluginManifestSetting, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%s: %w", path, errPathNotFound)
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
