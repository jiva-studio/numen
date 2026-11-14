// Package settings keeps the settings of the plugin that the connection requests read.
package settings

import (
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
)

const requestTimeoutSetting = "requestTimeout"

// Settings holds the values of the settings; the zero value holds none.
type Settings struct {
	mu             sync.RWMutex
	requestTimeout time.Duration
}

// Apply takes the settings the host sends.
func (s *Settings) Apply(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if timeout, ok := values[requestTimeoutSetting]; ok {
		s.requestTimeout = parseSeconds(timeout)
	}
}

// RequestTimeout returns how long a request of a connection may take, or zero when the setting
// holds no valid value.
func (s *Settings) RequestTimeout() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.requestTimeout
}

func parseSeconds(value string) time.Duration {
	seconds, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || math.IsNaN(seconds) || seconds <= 0 || math.IsInf(seconds, 0) || seconds >= float64(math.MaxInt64/int64(time.Second)) {
		return 0
	}
	return time.Duration(seconds * float64(time.Second))
}
