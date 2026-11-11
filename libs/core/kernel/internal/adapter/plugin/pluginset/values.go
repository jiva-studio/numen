package pluginset

import (
	"maps"
	"sync"
)

// Values keeps the values the user stores for the settings of the plugins, as the host has
// them in memory.
type Values struct {
	mu       sync.RWMutex
	byPlugin map[string]map[string]any
}

// NewValues returns the stored values of a host that has none.
func NewValues() *Values {
	return &Values{byPlugin: make(map[string]map[string]any)}
}

func (v *Values) Get(pluginID string) map[string]any {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return maps.Clone(v.byPlugin[pluginID])
}

func (v *Values) Merge(all map[string]map[string]any) {
	v.mu.Lock()
	defer v.mu.Unlock()
	for pluginID, values := range all {
		if _, ok := v.byPlugin[pluginID]; !ok {
			v.byPlugin[pluginID] = make(map[string]any)
		}
		maps.Copy(v.byPlugin[pluginID], values)
	}
}

func (v *Values) Replace(pluginID, key string, value any) (previous any, hadPrevious bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if _, ok := v.byPlugin[pluginID]; !ok {
		v.byPlugin[pluginID] = make(map[string]any)
	}
	previous, hadPrevious = v.byPlugin[pluginID][key]
	v.byPlugin[pluginID][key] = value
	return previous, hadPrevious
}

func (v *Values) Restore(pluginID, key string, previous any, hadPrevious bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if values, ok := v.byPlugin[pluginID]; ok && hadPrevious {
		values[key] = previous
	} else if ok {
		delete(values, key)
	}
}

func (v *Values) Fill(pluginID, key, value string) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	if hasValue(v.byPlugin[pluginID], key) {
		return false
	}
	if _, ok := v.byPlugin[pluginID]; !ok {
		v.byPlugin[pluginID] = make(map[string]any)
	}
	v.byPlugin[pluginID][key] = value
	return true
}

func (v *Values) Forget(pluginID, key, value string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if values, ok := v.byPlugin[pluginID]; ok && values[key] == value {
		delete(values, key)
	}
}

func hasValue(stored map[string]any, key string) bool {
	value, ok := stored[key]
	return ok && value != nil && value != ""
}
