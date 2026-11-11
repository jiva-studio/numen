// Package settings keeps the settings of the plugin the host hands to its backend.
package settings

import (
	"strings"
	"sync"
)

// Settings holds the values of the settings; the zero value holds none.
type Settings struct {
	mu             sync.RWMutex
	executablePath string
}

// Apply takes the settings the host sends.
func (s *Settings) Apply(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if execPath, ok := values["executablePath"]; ok {
		s.executablePath = strings.TrimSpace(execPath)
	}
}

// ExecutablePath returns the program the person set, or empty.
func (s *Settings) ExecutablePath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.executablePath
}
