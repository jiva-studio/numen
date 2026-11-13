// Package limits decides how long a request to a service may take. The limit is the setting of the
// plugin, else a default of the operation. The first request to a model of an OpenAI-compatible
// server has a long limit, because a local server loads the model while it answers.
package limits

import (
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	settingRequestTimeout = "requestTimeout"

	// typeCompatible is the connection type of a server the person runs, which loads a model on its first
	// request; operationComplete is the operation with the longest default limit.
	typeCompatible    = "openai-compatible"
	operationComplete = "complete"

	// DefaultCompleteLimit is how long a completion waits where the setting holds no value.
	DefaultCompleteLimit = 120 * time.Second
	// DefaultOtherLimit is how long any other operation waits where the setting holds no value.
	DefaultOtherLimit = 60 * time.Second
	// FirstRequestLimit bounds the first request to an OpenAI-compatible model.
	FirstRequestLimit = 10 * time.Minute
)

// Limits holds the request timeout of the settings and the models that have answered. The zero value is
// ready.
type Limits struct {
	mu      sync.Mutex
	timeout time.Duration
	served  map[string]struct{}
}

// Configure reads the settings of the plugin. A request timeout that is not a positive number in
// seconds leaves the default of the operation.
func (l *Limits) Configure(values map[string]string) {
	seconds, err := strconv.ParseFloat(strings.TrimSpace(values[settingRequestTimeout]), 64)
	var timeout time.Duration
	if err == nil && seconds > 0 && !math.IsInf(seconds, 0) && seconds < float64(math.MaxInt64/int64(time.Second)) {
		timeout = time.Duration(seconds * float64(time.Second))
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.timeout = timeout
}

// Get returns how long a request of the operation to the model of the type at the address may take.
func (l *Limits) Get(typeID, address, model, op string) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	if typeID == typeCompatible {
		if _, isServed := l.served[servedKey(address, model)]; !isServed {
			return FirstRequestLimit
		}
	}
	if l.timeout > 0 {
		return l.timeout
	}
	if op == operationComplete {
		return DefaultCompleteLimit
	}
	return DefaultOtherLimit
}

// MarkServed records that the model at the address has answered, which ends its long first limit.
func (l *Limits) MarkServed(address, model string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.served == nil {
		l.served = map[string]struct{}{}
	}
	l.served[servedKey(address, model)] = struct{}{}
}

func servedKey(address, model string) string {
	return address + "\x00" + model
}
