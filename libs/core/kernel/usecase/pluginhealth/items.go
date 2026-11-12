// Package pluginhealth keeps what the plugins and the core say about whether each plugin works: one
// list of health items for the whole installation, which the windows list and watch.
package pluginhealth

import (
	"context"
	"maps"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// Items is the health items of every plugin, in memory. It outlives the processes of the plugins:
// an idle stop keeps the items, and only clearing a plugin removes them. It is safe to use from
// several goroutines.
type Items struct {
	mu      sync.Mutex
	items   map[itemID]plugin.HealthItem
	order   []itemID
	waiting []chan []plugin.HealthItem
}

type itemID struct {
	pluginID string
	key      string
}

// NewItems returns an empty list.
func NewItems() *Items { return &Items{items: map[itemID]plugin.HealthItem{}} }

// Set puts the item in the list, or replaces the one of the plugin under its key. Items are listed in
// the order they were first set.
func (s *Items) Set(item plugin.HealthItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setLocked(item)
}

// SetWithin sets the item as Set does unless it is a new key of a plugin that reported maxKeys items of
// its own already, and reports whether it set it. The items of the core are not counted.
func (s *Items) SetWithin(item plugin.HealthItem, maxKeys int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, isListed := s.items[itemID{pluginID: item.PluginID, key: item.Key}]; !isListed && s.countOwnLocked(item.PluginID) >= maxKeys {
		return false
	}
	s.setLocked(item)
	return true
}

func (s *Items) setLocked(item plugin.HealthItem) {
	id := itemID{pluginID: item.PluginID, key: item.Key}
	previous, isListed := s.items[id]
	if isListed && isSameItem(previous, item) {
		return
	}
	if !isListed {
		s.order = append(s.order, id)
	}
	s.items[id] = cloneItem(item)
	s.notifyLocked()
}

func (s *Items) countOwnLocked(pluginID string) int {
	count := 0
	for id := range s.items {
		if id.pluginID == pluginID && !plugin.IsCoreHealthKey(id.key) {
			count++
		}
	}
	return count
}

// Remove takes the item of the plugin under the key out of the list.
func (s *Items) Remove(pluginID, key string) {
	s.removeWhere(func(id itemID) bool { return id.pluginID == pluginID && id.key == key })
}

// RemovePlugin takes every item of the plugin out of the list.
func (s *Items) RemovePlugin(pluginID string) {
	s.removeWhere(func(id itemID) bool { return id.pluginID == pluginID })
}

// RemoveAll empties the list.
func (s *Items) RemoveAll() {
	s.removeWhere(func(itemID) bool { return true })
}

// List returns the items of every plugin.
func (s *Items) List() []plugin.HealthItem {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.listLocked()
}

// Watch returns the list now and every time it changes, until the context ends. A listener that is not
// keeping up is given the newest list and not a queue of old ones.
func (s *Items) Watch(ctx context.Context) <-chan []plugin.HealthItem {
	watcher := make(chan []plugin.HealthItem, 1)
	s.mu.Lock()
	s.waiting = append(s.waiting, watcher)
	watcher <- s.listLocked()
	s.mu.Unlock()

	go func() {
		<-ctx.Done()
		s.mu.Lock()
		defer s.mu.Unlock()
		s.waiting = slices.DeleteFunc(s.waiting, func(one chan []plugin.HealthItem) bool { return one == watcher })
		close(watcher)
	}()
	return watcher
}

func (s *Items) removeWhere(isRemoved func(itemID) bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := len(s.items)
	maps.DeleteFunc(s.items, func(id itemID, _ plugin.HealthItem) bool { return isRemoved(id) })
	if len(s.items) == count {
		return
	}
	s.order = slices.DeleteFunc(s.order, isRemoved)
	s.notifyLocked()
}

func (s *Items) listLocked() []plugin.HealthItem {
	out := make([]plugin.HealthItem, 0, len(s.order))
	for _, id := range s.order {
		out = append(out, cloneItem(s.items[id]))
	}
	return out
}

func (s *Items) notifyLocked() {
	list := s.listLocked()
	for _, watcher := range s.waiting {
		select {
		case <-watcher:
		default:
		}
		select {
		case watcher <- list:
		default:
		}
	}
}

func cloneItem(item plugin.HealthItem) plugin.HealthItem {
	if item.Action != nil {
		action := *item.Action
		action.Values = maps.Clone(action.Values)
		item.Action = &action
	}
	return item
}

func isSameItem(a, b plugin.HealthItem) bool {
	if a.Severity != b.Severity || a.Text != b.Text || a.Detail != b.Detail || (a.Action == nil) != (b.Action == nil) {
		return false
	}
	if a.Action == nil {
		return true
	}
	return a.Action.Command == b.Action.Command && a.Action.Title == b.Action.Title && maps.Equal(a.Action.Values, b.Action.Values)
}
