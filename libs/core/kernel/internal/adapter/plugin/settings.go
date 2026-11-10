package plugin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// tokenBytes is the number of random bytes in a generated token.
const tokenBytes = 32

// generatedSetting is one value the host generated for a setting of a plugin.
type generatedSetting struct {
	pluginID string
	key      string
	value    string
}

// Settings keeps in memory the values the user stores for the settings of the plugins, and the
// performance profile. Writing a value to disk, and what follows from a change, are not done here.
type Settings struct {
	t         *table
	errs      *errorSink
	listeners *Listeners
	config    *ConfigPush

	mu          sync.RWMutex
	persist     plugindomain.PersistHandler
	performance string
}

func newSettings(t *table, errs *errorSink, listeners *Listeners) *Settings {
	return &Settings{t: t, errs: errs, listeners: listeners}
}

// SetPersistHandler registers a callback that writes plugin settings changes into core configuration.
func (s *Settings) SetPersistHandler(handler plugindomain.PersistHandler) {
	s.mu.Lock()
	s.persist = handler
	s.mu.Unlock()
	s.FillGeneratedSettings()
}

// SetSettings loads persisted plugin configurations into host memory.
func (s *Settings) SetSettings(all map[string]map[string]any) {
	s.t.mu.Lock()
	for pluginID, cfg := range all {
		if _, ok := s.t.settings[pluginID]; !ok {
			s.t.settings[pluginID] = make(map[string]any)
		}
		maps.Copy(s.t.settings[pluginID], cfg)
	}
	s.t.mu.Unlock()
	s.FillGeneratedSettings()
	s.listeners.syncAll(context.Background())
}

// GetPluginSettings returns resolved settings for a plugin with defaults overlaid by stored values.
func (s *Settings) GetPluginSettings(pluginID string) map[string]any {
	s.t.mu.RLock()
	defer s.t.mu.RUnlock()
	return s.t.getPluginSettingsLocked(pluginID)
}

// GetPluginConfigJSON returns serialized JSON configuration for the specified plugin.
func (s *Settings) GetPluginConfigJSON(pluginID string) (string, error) {
	data, err := json.Marshal(s.GetPluginSettings(pluginID))
	if err != nil {
		return "", fmt.Errorf("marshal plugin %s settings: %w", pluginID, err)
	}
	return string(data), nil
}

// ReplaceSetting stores the value of one setting in memory. It returns the value it replaced, whether
// there was one, and whether the dependencies the plugin declares are others now.
func (s *Settings) ReplaceSetting(pluginID, key string, value any) (previous any, hadPrevious, changesDependencies bool) {
	s.t.mu.Lock()
	defer s.t.mu.Unlock()
	before := s.t.getRequirementIDsLocked(pluginID)
	if _, ok := s.t.settings[pluginID]; !ok {
		s.t.settings[pluginID] = make(map[string]any)
	}
	previous, hadPrevious = s.t.settings[pluginID][key]
	s.t.settings[pluginID][key] = value
	return previous, hadPrevious, !slices.Equal(before, s.t.getRequirementIDsLocked(pluginID))
}

// RestoreSetting puts back the value ReplaceSetting replaced, or removes the setting where there was none.
func (s *Settings) RestoreSetting(pluginID, key string, previous any, hadPrevious bool) {
	s.t.mu.Lock()
	defer s.t.mu.Unlock()
	if values, ok := s.t.settings[pluginID]; ok && hadPrevious {
		values[key] = previous
	} else if ok {
		delete(values, key)
	}
}

// SetPerformanceProfile records the profile of the performance setting and pushes it to every
// running plugin when it changed. A plugin that is not running receives it with its first
// Configure call.
func (s *Settings) SetPerformanceProfile(ctx context.Context, profile string) {
	s.mu.Lock()
	if s.performance == profile {
		s.mu.Unlock()
		return
	}
	s.performance = profile
	s.mu.Unlock()

	s.t.mu.RLock()
	ids := make([]string, 0, len(s.t.endpoints))
	for id := range s.t.endpoints {
		ids = append(ids, id)
	}
	s.t.mu.RUnlock()

	for _, id := range ids {
		if err := s.config.NotifyPluginConfig(ctx, id); err != nil {
			s.errs.report(fmt.Errorf("push the performance profile to plugin %s: %w", id, err))
		}
	}
}

func (s *Settings) getPerformanceProfile() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.performance
}

// FillGeneratedSettings gives every enabled plugin a value for each setting that is generated and
// holds none, and persists it.
func (s *Settings) FillGeneratedSettings() {
	s.mu.RLock()
	persist := s.persist
	s.mu.RUnlock()
	if persist == nil {
		return
	}
	s.t.mu.Lock()
	var filled []generatedSetting
	var failed []error
	for pluginID, p := range s.t.plugins {
		if s.t.isPluginEnabledLocked(pluginID) {
			values, errs := s.storeGeneratedSettingsLocked(pluginID, p.Manifest.Settings)
			filled = append(filled, values...)
			failed = append(failed, errs...)
		}
	}
	s.t.mu.Unlock()

	for _, one := range filled {
		if err := persist(one.pluginID, one.key, one.value); err != nil {
			s.forgetGeneratedSetting(one)
			failed = append(failed, fmt.Errorf("persist generated setting %s of plugin %s: %w", one.key, one.pluginID, err))
		}
	}
	for _, err := range failed {
		s.errs.report(err)
	}
}

func (s *Settings) storeGeneratedSettingsLocked(pluginID string, settings []manifest.Setting) ([]generatedSetting, []error) {
	var filled []generatedSetting
	var failed []error
	for _, one := range settings {
		if one.Generate != manifest.SettingGenerateToken || hasValue(s.t.settings[pluginID], one.Key) {
			continue
		}
		token, err := createToken()
		if err != nil {
			failed = append(failed, fmt.Errorf("generate setting %s of plugin %s: %w", one.Key, pluginID, err))
			continue
		}
		if _, ok := s.t.settings[pluginID]; !ok {
			s.t.settings[pluginID] = make(map[string]any)
		}
		s.t.settings[pluginID][one.Key] = token
		filled = append(filled, generatedSetting{pluginID: pluginID, key: one.Key, value: token})
	}
	return filled, failed
}

func (s *Settings) forgetGeneratedSetting(one generatedSetting) {
	s.t.mu.Lock()
	defer s.t.mu.Unlock()
	if values, ok := s.t.settings[one.pluginID]; ok && values[one.key] == one.value {
		delete(values, one.key)
	}
}

func hasValue(stored map[string]any, key string) bool {
	value, ok := stored[key]
	return ok && value != nil && value != ""
}

func createToken() (string, error) {
	raw := make([]byte, tokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return hex.EncodeToString(raw), nil
}

// The reasons a setting of type file does not accept a path.
var (
	errPathNotFound = errors.New("not found")
	errNotAFile     = errors.New("not a file")
	errNotAccepted  = errors.New("wrong type of file")
)

// CheckPluginSetting reports why a setting the plugin declares does not accept the value. A setting
// of type file accepts an empty value or an existing file with an extension it accepts. The port
// setting of a listener accepts a port the host can bind. Any other setting accepts any value.
func (s *Settings) CheckPluginSetting(pluginID, key string, value any) error {
	if err := s.listeners.checkPort(context.Background(), pluginID, key, value); err != nil {
		return err
	}
	s.t.mu.RLock()
	p, ok := s.t.plugins[pluginID]
	s.t.mu.RUnlock()
	if !ok {
		return nil
	}
	declared, ok := p.Manifest.GetSetting(key)
	if !ok || declared.Type != manifest.SettingTypeFile {
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

func checkPath(declared manifest.Setting, path string) error {
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
