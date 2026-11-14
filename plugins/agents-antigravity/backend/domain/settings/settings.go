// Package settings keeps the settings of the plugin the host hands to its backend.
package settings

import (
	"encoding/json"
	"math"
	"strings"
	"sync"
	"time"
)

// Settings holds the values of the settings; the zero value holds none.
type Settings struct {
	mu             sync.RWMutex
	executablePath string
	requestTimeout time.Duration
}

// Apply takes the settings the host sends.
func (s *Settings) Apply(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if execPath, ok := values["executablePath"]; ok {
		s.executablePath = strings.TrimSpace(execPath)
	}
	if timeout, ok := values["requestTimeout"]; ok {
		s.requestTimeout = parseSeconds(timeout)
	}
}

func parseSeconds(value string) time.Duration {
	var seconds float64
	if err := json.Unmarshal([]byte(strings.TrimSpace(value)), &seconds); err != nil || math.Signbit(seconds) || seconds >= float64(math.MaxInt64/int64(time.Second)) {
		return 0
	}
	return time.Duration(seconds * float64(time.Second))
}

// ExecutablePath returns the program the person set, or empty.
func (s *Settings) ExecutablePath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.executablePath
}

// RequestTimeout returns how long a request of a connection may take, or zero when the setting holds no valid value.
func (s *Settings) RequestTimeout() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.requestTimeout
}
