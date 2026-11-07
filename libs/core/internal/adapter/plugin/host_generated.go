package plugin

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

// tokenBytes is the number of random bytes in a generated token.
const tokenBytes = 32

// generatedSetting is one value the host generated for a setting of a plugin.
type generatedSetting struct {
	pluginID string
	key      string
	value    string
}

// fillGeneratedSettings gives every setting of an enabled plugin that asks for a generated value and
// holds none or "" a new value, in memory first and then through the persist handler. A value
// already held is kept, so a call made again while persisting generates nothing.
func (h *Host) fillGeneratedSettings() {
	h.mu.Lock()
	persist := h.persistHandler
	if persist == nil {
		h.mu.Unlock()
		return
	}
	var filled []generatedSetting
	var failed []error
	for pluginID, p := range h.plugins {
		if h.isPluginEnabledLocked(pluginID) {
			values, errs := h.storeGeneratedSettingsLocked(pluginID, p.Manifest.Settings)
			filled = append(filled, values...)
			failed = append(failed, errs...)
		}
	}
	h.mu.Unlock()

	for _, one := range filled {
		if err := persist(one.pluginID, one.key, one.value); err != nil {
			h.forgetGeneratedSetting(one)
			failed = append(failed, fmt.Errorf("persist generated setting %s of plugin %s: %w", one.key, one.pluginID, err))
		}
	}
	for _, err := range failed {
		h.reportError(err)
	}
}

// storeGeneratedSettingsLocked stores a new token in memory for each of the settings that asks for
// one and holds no value, and returns the tokens stored.
func (h *Host) storeGeneratedSettingsLocked(pluginID string, settings []manifest.Setting) ([]generatedSetting, []error) {
	var filled []generatedSetting
	var failed []error
	for _, s := range settings {
		if s.Generate != manifest.SettingGenerateToken || hasValue(h.settings[pluginID], s.Key) {
			continue
		}
		token, err := createToken()
		if err != nil {
			failed = append(failed, fmt.Errorf("generate setting %s of plugin %s: %w", s.Key, pluginID, err))
			continue
		}
		if _, ok := h.settings[pluginID]; !ok {
			h.settings[pluginID] = make(map[string]any)
		}
		h.settings[pluginID][s.Key] = token
		filled = append(filled, generatedSetting{pluginID: pluginID, key: s.Key, value: token})
	}
	return filled, failed
}

// forgetGeneratedSetting removes a generated value that could not be persisted, unless the setting
// has changed since.
func (h *Host) forgetGeneratedSetting(one generatedSetting) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if values, ok := h.settings[one.pluginID]; ok && values[one.key] == one.value {
		delete(values, one.key)
	}
}

// hasValue reports whether the stored values hold the key with a value other than "".
func hasValue(stored map[string]any, key string) bool {
	value, ok := stored[key]
	return ok && value != nil && value != ""
}

// createToken returns tokenBytes random bytes in lowercase hex.
func createToken() (string, error) {
	raw := make([]byte, tokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return hex.EncodeToString(raw), nil
}
