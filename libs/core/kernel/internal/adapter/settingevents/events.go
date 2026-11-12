package settingevents

import (
	"maps"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/watchhub"
)

// Source is where the events come from.
type Source struct {
	Read    func() (settings.Values, error)
	Initial func() settings.Values
	Report  func(error)
	Hub     *watchhub.Hub
	Path    string
}

// Events delivers the settings that change to the readers subscribed. The file is watched while
// somebody is subscribed.
type Events struct {
	mu          sync.Mutex
	subscribers map[*subscriber]struct{}
	values      settings.Values
	source      Source
	stop        func()
}

// New returns events with no reader.
func New() *Events {
	return &Events{subscribers: map[*subscriber]struct{}{}}
}

// Subscribe starts delivering to a new reader. The first reader starts the watch of the file. The
// returned function stops the delivery and closes the channel.
func (e *Events) Subscribe(source Source) (<-chan []settings.Setting, func()) {
	one := newSubscriber()
	e.mu.Lock()
	if len(e.subscribers) == 0 {
		e.begin(source)
	}
	e.subscribers[one] = struct{}{}
	e.mu.Unlock()
	go one.deliver()

	return one.out, func() {
		one.once.Do(func() {
			close(one.leave)
			e.mu.Lock()
			defer e.mu.Unlock()
			delete(e.subscribers, one)
			if len(e.subscribers) == 0 {
				e.end()
			}
		})
	}
}

func (e *Events) begin(source Source) {
	e.source = source
	e.stop = watchFile(source.Hub, source.Path, source.Report, e.refreshAfterSave)
	values, err := source.Read()
	if err != nil {
		source.Report(err)
		values = source.Initial()
	}
	e.values = values
}

func (e *Events) end() {
	if e.stop != nil {
		e.stop()
	}
	e.stop, e.source, e.values = nil, Source{}, nil
}

func (e *Events) refreshAfterSave() {
	e.mu.Lock()
	read := e.source.Read
	e.mu.Unlock()
	if read != nil {
		e.Refresh(read)
	}
}

// Refresh reads the values in force and delivers every setting that differs from what the readers
// were last told, as one batch. A file that cannot be read changes nothing.
func (e *Events) Refresh(read func() (settings.Values, error)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.values == nil {
		return
	}
	now, err := read()
	if err != nil {
		e.source.Report(err)
		return
	}
	var batch []settings.Setting
	for _, key := range slices.Sorted(maps.Keys(merge(e.values, now))) {
		after, isSet := now[key]
		if !isSet {
			after = removed
		}
		if before, wasSet := e.values[key]; before != after || wasSet != isSet {
			batch = append(batch, settings.Setting{Path: settings.SplitPath(key), JSON: after})
		}
	}
	e.values = now
	for one := range e.subscribers {
		if len(batch) > 0 {
			one.push(slices.Clone(batch))
		}
	}
}

// removed is the value of a setting that no longer holds one: the null that resets a setting.
const removed = "null"

func merge(first, second settings.Values) map[string]struct{} {
	keys := make(map[string]struct{}, len(first)+len(second))
	for key := range first {
		keys[key] = struct{}{}
	}
	for key := range second {
		keys[key] = struct{}{}
	}
	return keys
}
