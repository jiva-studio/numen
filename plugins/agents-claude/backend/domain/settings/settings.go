// Package settings keeps the settings of the plugin the host hands to its backend.
package settings

import (
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
)

// readsHooksSetting is the setting of the plugin that lets the program read the hooks, skills and
// standing instructions of the person.
const readsHooksSetting = "readsHooksAndSkills"

// requestTimeoutSetting is the setting of the plugin that limits a request of a connection, in seconds.
const requestTimeoutSetting = "requestTimeout"

// Values are the settings in force.
type Values struct {
	ExecutablePath string
	MayReadHooks   bool
	RequestTimeout time.Duration
}

// Settings holds the values of the settings; the zero value holds none.
type Settings struct {
	mu     sync.RWMutex
	values Values
}

// Apply takes the settings the host sends.
func (s *Settings) Apply(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if execPath, ok := values["executablePath"]; ok {
		s.values.ExecutablePath = strings.TrimSpace(execPath)
	}
	if reads, ok := values[readsHooksSetting]; ok {
		s.values.MayReadHooks = reads == "true"
	}
	if timeout, ok := values[requestTimeoutSetting]; ok {
		s.values.RequestTimeout = parseSeconds(timeout)
	}
}

func parseSeconds(value string) time.Duration {
	seconds, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || math.IsNaN(seconds) || seconds <= 0 || math.IsInf(seconds, 0) || seconds >= float64(math.MaxInt64/int64(time.Second)) {
		return 0
	}
	return time.Duration(seconds * float64(time.Second))
}

// Get returns the values in force.
func (s *Settings) Get() Values {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.values
}
