package settings_test

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// stored is a settings file held in memory, by the dotted path of each value.
type stored struct {
	values      map[string]any
	connections string
}

func (s stored) GetValue(path ...string) (any, bool) {
	value, found := s.values[strings.Join(path, ".")]
	return value, found
}

func (s stored) GetConnectionsJSON() (string, error) { return s.connections, nil }

// memoryFile is the settings file as the use cases see it, and records what happens to it.
type memoryFile struct {
	mu      sync.Mutex
	current stored
	writes  []settingsdomain.Setting
	events  *[]string
	failure error
}

func (f *memoryFile) ReadChecked() (settingsdomain.Stored, error) { return f.current, f.failure }

func (f *memoryFile) ReadUnchecked() (settingsdomain.Stored, error) { return f.current, f.failure }

func (f *memoryFile) Write(entries []settingsdomain.Setting) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.events != nil {
		*f.events = append(*f.events, "write file")
	}
	f.writes = append(f.writes, entries...)
	return f.failure
}

func (f *memoryFile) GetDefaults() settingsdomain.Stored { return stored{values: map[string]any{}} }

// declare returns declarations of a setting of no type at each path, which takes any value. Every other
// key is undeclared.
func declare(paths ...[]string) settingsdomain.Declarations {
	return func(owner string) ([]manifest.Setting, bool) {
		var declared []manifest.Setting
		for _, one := range paths {
			if one[0] == owner {
				declared = append(declared, manifest.Setting{Key: strings.Join(one[1:], ".")})
			}
		}
		return declared, true
	}
}

type refreshRecorder struct{ events *[]string }

func (r refreshRecorder) Refresh() { *r.events = append(*r.events, "refresh events") }

// pluginHost records the calls the plugin host receives.
type pluginHost struct {
	mu       sync.Mutex
	updates  []string
	checks   []string
	profiles []string
	settings map[string]map[string]any
	persist  plugin.PersistHandler
	reason   error
}

func (h *pluginHost) UpdatePluginSetting(_ context.Context, pluginID, key string, _ any) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.updates = append(h.updates, pluginID+"."+key)
	return nil
}

func (h *pluginHost) CheckPluginSetting(pluginID, key string, _ any) error {
	h.checks = append(h.checks, pluginID+"."+key)
	return h.reason
}

func (h *pluginHost) SetSettings(all map[string]map[string]any) { h.settings = all }

func (h *pluginHost) SetPerformanceProfile(_ context.Context, profile string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.profiles = append(h.profiles, profile)
}

func (h *pluginHost) SetPersistHandler(handler plugin.PersistHandler) { h.persist = handler }

type appDeclarations []manifest.Setting

func (d appDeclarations) GetAppDeclaration() []manifest.Setting { return d }

type pluginInfos []plugin.Info

func (p pluginInfos) GetPluginInfos() []plugin.Info { return p }

type jsonCodec struct{}

func (jsonCodec) Decode(text string) (any, error) {
	var value any
	err := json.Unmarshal([]byte(text), &value)
	return value, err
}

func (jsonCodec) Encode(value any) (string, error) {
	encoded, err := json.Marshal(value)
	return string(encoded), err
}
