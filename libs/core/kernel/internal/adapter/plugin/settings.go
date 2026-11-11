package plugin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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

// portChecker says why a port is not one a listener of a plugin may bind, and binds the ports the settings ask for.
type portChecker interface {
	checkPort(ctx context.Context, pluginID, key string, value any) error
	syncAll(ctx context.Context)
}

// profilePusher gives the running plugins the performance profile.
type profilePusher interface {
	SetPerformanceProfile(ctx context.Context, profile string)
}

// settingPlugins is what the settings read of the set of plugins.
type settingPlugins interface {
	getPluginSettings(pluginID string) map[string]any
	getRequirementIDs(pluginID string) []string
	listEnabled() []manifest.Discovered
	getKnown(pluginID string) (manifest.Discovered, bool)
}

// Settings keeps in memory the values the user stores for the settings of the plugins. Writing a
// value to disk, and what follows from a change, are not done here.
type Settings struct {
	values    *settingValues
	set       settingPlugins
	listeners portChecker
	profile   profilePusher
	report    func(error)

	mu      sync.RWMutex
	persist plugindomain.PersistHandler
}

func newSettings(values *settingValues, set settingPlugins, listeners portChecker, profile profilePusher, report func(error)) *Settings {
	return &Settings{values: values, set: set, listeners: listeners, profile: profile, report: report}
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
	s.values.merge(all)
	s.FillGeneratedSettings()
	s.listeners.syncAll(context.Background())
}

// SetPerformanceProfile records the profile of the performance setting and pushes it to every
// running plugin when it changed.
func (s *Settings) SetPerformanceProfile(ctx context.Context, profile string) {
	s.profile.SetPerformanceProfile(ctx, profile)
}

// GetPluginSettings returns resolved settings for a plugin with defaults overlaid by stored values.
func (s *Settings) GetPluginSettings(pluginID string) map[string]any {
	return s.set.getPluginSettings(pluginID)
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
	before := s.set.getRequirementIDs(pluginID)
	previous, hadPrevious = s.values.replace(pluginID, key, value)
	return previous, hadPrevious, !slices.Equal(before, s.set.getRequirementIDs(pluginID))
}

// RestoreSetting puts back the value ReplaceSetting replaced, or removes the setting where there was none.
func (s *Settings) RestoreSetting(pluginID, key string, previous any, hadPrevious bool) {
	s.values.restore(pluginID, key, previous, hadPrevious)
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
	filled, failed := s.storeGeneratedSettings()
	for _, one := range filled {
		if err := persist(one.pluginID, one.key, one.value); err != nil {
			s.values.forget(one.pluginID, one.key, one.value)
			failed = append(failed, fmt.Errorf("persist generated setting %s of plugin %s: %w", one.key, one.pluginID, err))
		}
	}
	for _, err := range failed {
		s.report(err)
	}
}

func (s *Settings) storeGeneratedSettings() ([]generatedSetting, []error) {
	var filled []generatedSetting
	var failed []error
	for _, p := range s.set.listEnabled() {
		for _, one := range p.Manifest.Settings {
			if one.Generate != manifest.SettingGenerateToken {
				continue
			}
			token, err := createToken()
			if err != nil {
				failed = append(failed, fmt.Errorf("generate setting %s of plugin %s: %w", one.Key, p.Manifest.ID, err))
				continue
			}
			if s.values.fill(p.Manifest.ID, one.Key, token) {
				filled = append(filled, generatedSetting{pluginID: p.Manifest.ID, key: one.Key, value: token})
			}
		}
	}
	return filled, failed
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
	p, ok := s.set.getKnown(pluginID)
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
