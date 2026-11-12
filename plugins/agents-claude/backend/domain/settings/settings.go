// Package settings keeps the settings of the plugin the host hands to its backend.
package settings

import (
	"strings"
	"sync"
)

// readsHooksSetting is the setting of the plugin that lets the program read the hooks, skills and
// standing instructions of the person.
const readsHooksSetting = "readsHooksAndSkills"

// Values are the settings in force.
type Values struct {
	DefaultModel   string
	ExecutablePath string
	MayReadHooks   bool
}

// Settings holds the values of the settings; the zero value holds none.
type Settings struct {
	mu     sync.RWMutex
	values Values
}

// Apply takes the settings the host sends. A model that is empty leaves the model in force.
func (s *Settings) Apply(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if model, ok := values["model"]; ok && model != "" {
		s.values.DefaultModel = model
	}
	if execPath, ok := values["executablePath"]; ok {
		s.values.ExecutablePath = strings.TrimSpace(execPath)
	}
	if reads, ok := values[readsHooksSetting]; ok {
		s.values.MayReadHooks = reads == "true"
	}
}

// Get returns the values in force.
func (s *Settings) Get() Values {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.values
}
