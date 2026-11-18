// Package logs records structured log entries to the in-memory buffer and the daily file store,
// and streams them to watchers.
package logs

import (
	"context"
	"maps"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/logs"
)

// MaxBufferedEntries is the number of recent log entries kept in memory for watchers.
const MaxBufferedEntries = 1000

// Entries is the recent log entries of the installation, in memory. It is safe to use from several
// goroutines.
type Entries struct {
	mu      sync.Mutex
	items   []logs.Entry
	waiting []chan []logs.Entry
}

// NewEntries returns an empty buffer of recent log entries.
func NewEntries() *Entries {
	return &Entries{}
}

// Append adds one entry to the buffer, trimming the oldest entry when the buffer exceeds
// MaxBufferedEntries, and sends the new entry to every active watcher.
func (s *Entries) Append(entry logs.Entry) {
	cloned := cloneEntry(entry)
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.items) >= MaxBufferedEntries {
		s.items = slices.Clone(s.items[len(s.items)-MaxBufferedEntries+1:])
	}
	s.items = append(s.items, cloned)
	batch := []logs.Entry{cloneEntry(cloned)}
	for _, watcher := range s.waiting {
		sendBatch(watcher, batch)
	}
}

// Watch returns a channel that receives the buffered entries (merged with initial if non-empty) at
// once and each new batch of entries as they are appended, until ctx ends.
func (s *Entries) Watch(ctx context.Context, initial []logs.Entry) <-chan []logs.Entry {
	watcher := make(chan []logs.Entry, 16)
	s.mu.Lock()
	s.waiting = append(s.waiting, watcher)
	snapshot := mergeInitial(initial, s.items)
	if len(snapshot) > 0 {
		watcher <- snapshot
	}
	s.mu.Unlock()

	go func() {
		<-ctx.Done()
		s.mu.Lock()
		defer s.mu.Unlock()
		s.waiting = slices.DeleteFunc(s.waiting, func(one chan []logs.Entry) bool { return one == watcher })
		close(watcher)
	}()
	return watcher
}

func mergeInitial(disk, memory []logs.Entry) []logs.Entry {
	if len(disk) == 0 && len(memory) == 0 {
		return nil
	}
	if len(disk) == 0 {
		return cloneEntries(memory)
	}
	if len(memory) == 0 {
		return cloneEntries(disk)
	}
	firstMemoryTime := memory[0].Time
	var prefix []logs.Entry
	for _, one := range disk {
		if one.Time.Before(firstMemoryTime) {
			prefix = append(prefix, cloneEntry(one))
		}
	}
	merged := slices.Concat(prefix, cloneEntries(memory))
	if len(merged) > MaxBufferedEntries {
		merged = merged[len(merged)-MaxBufferedEntries:]
	}
	return merged
}

func sendBatch(watcher chan []logs.Entry, batch []logs.Entry) {
	select {
	case watcher <- batch:
	default:
		select {
		case stale := <-watcher:
			combined := append(slices.Clone(stale), batch...)
			if len(combined) > MaxBufferedEntries {
				combined = combined[len(combined)-MaxBufferedEntries:]
			}
			watcher <- combined
		default:
		}
	}
}

func cloneEntries(list []logs.Entry) []logs.Entry {
	out := make([]logs.Entry, len(list))
	for i, one := range list {
		out[i] = cloneEntry(one)
	}
	return out
}

func cloneEntry(entry logs.Entry) logs.Entry {
	entry.Properties = maps.Clone(entry.Properties)
	return entry
}
