package container

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/rjeczalik/notify"

	"github.com/jiva-studio/numen/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/adapter/settings/hardware"
	"github.com/jiva-studio/numen/libs/core/port/settingsport"
)

// settingSaveDelay is how long the settings file is left alone after an event of the system
// before it is read. A save is several events: the file is truncated, written and renamed.
const settingSaveDelay = 100 * time.Millisecond

// settingEventBacklog is how many events of the system wait for the debounce.
const settingEventBacklog = 64

// settingValues is the value in force of every declared setting as JSON, by its path joined with
// a zero byte. A setting with neither a stored value nor a default is left out.
type settingValues map[string]string

func joinSettingPath(path []string) string { return strings.Join(path, "\x00") }

// settingEvents delivers the settings that change to everyone who subscribed. The settings file is
// watched while somebody is subscribed.
type settingEvents struct {
	mu          sync.Mutex
	subscribers map[*settingSubscriber]struct{}
	// values is what the subscribers were last told. It is nil while nobody is subscribed.
	values settingValues
	// read, report and stopWatch belong to the subscription that started the watch.
	read      func() (settingValues, error)
	report    func(error)
	stopWatch func()

	hostOnce sync.Once
}

func newSettingEvents() *settingEvents {
	return &settingEvents{subscribers: map[*settingSubscriber]struct{}{}}
}

// settingSubscriber is one reader. Its batches wait in a queue, so a reader that is slow to take
// them never holds up the write that made them.
type settingSubscriber struct {
	out  chan []settingsport.Setting
	done chan struct{}
	wake chan struct{}
	once sync.Once

	mu    sync.Mutex
	queue [][]settingsport.Setting
}

func (s *settingSubscriber) push(batch []settingsport.Setting) {
	s.mu.Lock()
	s.queue = append(s.queue, batch)
	s.mu.Unlock()
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *settingSubscriber) next() ([]settingsport.Setting, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.queue) == 0 {
		return nil, false
	}
	batch := s.queue[0]
	s.queue = s.queue[1:]
	return batch, true
}

// deliver hands the queued batches over, in order, until the subscriber stops.
func (s *settingSubscriber) deliver() {
	defer close(s.out)
	for {
		batch, found := s.next()
		if !found {
			select {
			case <-s.wake:
				continue
			case <-s.done:
				return
			}
		}
		select {
		case s.out <- batch:
		case <-s.done:
			return
		}
	}
}

// subscribe starts delivering to a new reader. The first reader starts the watch of the file.
func (e *settingEvents) subscribe(c Config) (<-chan []settingsport.Setting, func()) {
	one := &settingSubscriber{
		out:  make(chan []settingsport.Setting),
		done: make(chan struct{}),
		wake: make(chan struct{}, 1),
	}
	e.mu.Lock()
	if len(e.subscribers) == 0 {
		e.start(c)
	}
	e.subscribers[one] = struct{}{}
	e.mu.Unlock()
	go one.deliver()

	return one.out, func() {
		one.once.Do(func() {
			close(one.done)
			e.mu.Lock()
			defer e.mu.Unlock()
			delete(e.subscribers, one)
			if len(e.subscribers) == 0 {
				e.finish()
			}
		})
	}
}

// start watches the file and takes the values the first reader starts from. The caller holds mu.
func (e *settingEvents) start(c Config) {
	e.read, e.report = c.readSettingValues, c.handleError
	e.stopWatch = c.watchSettingsFile(e.refreshFromFile)
	values, err := e.read()
	if err != nil {
		e.report(err)
		values, _ = c.listSettingValues(DefaultSettings())
	}
	e.values = values
}

// finish stops the watch once the last reader left. The caller holds mu.
func (e *settingEvents) finish() {
	if e.stopWatch != nil {
		e.stopWatch()
	}
	e.stopWatch, e.read, e.report, e.values = nil, nil, nil, nil
}

func (e *settingEvents) refreshFromFile() {
	e.mu.Lock()
	read := e.read
	e.mu.Unlock()
	if read != nil {
		e.refresh(read)
	}
}

// refresh reads the values in force and delivers every setting that differs from what the readers
// were last told, as one batch. A file that cannot be read changes nothing.
func (e *settingEvents) refresh(read func() (settingValues, error)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.values == nil {
		return
	}
	now, err := read()
	if err != nil {
		e.report(err)
		return
	}
	var batch []settingsport.Setting
	for _, key := range slices.Sorted(maps.Keys(now)) {
		if e.values[key] != now[key] {
			batch = append(batch, settingsport.Setting{Path: strings.Split(key, "\x00"), JSON: now[key]})
		}
	}
	e.values = now
	if len(batch) == 0 {
		return
	}
	for one := range e.subscribers {
		one.push(slices.Clone(batch))
	}
}

// SubscribeSettingChanges starts listening to the settings that change, whether a window, a plugin
// or the palette writes them or a person edits the file by hand. See settingsport.ChangeSubscriber.
func (c Config) SubscribeSettingChanges() (<-chan []settingsport.Setting, func()) {
	events := c.getResources().settingEvents
	if events == nil {
		closed := make(chan []settingsport.Setting)
		close(closed)
		return closed, func() {}
	}
	return events.subscribe(c)
}

// refreshSettingEvents delivers what a write changed.
func (c Config) refreshSettingEvents() {
	if events := c.getResources().settingEvents; events != nil {
		events.refresh(c.readSettingValues)
	}
}

// readSettingValues reads the values in force from the settings file. A file that is not there is
// an unconfigured installation, and reading does not write it.
func (c Config) readSettingValues() (settingValues, error) {
	path, err := c.settingsFile()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return c.listSettingValues(DefaultSettings())
	}
	target, err := c.getCheckedSettingsAt(path)
	if err != nil {
		return nil, err
	}
	return c.listSettingValues(target)
}

// listSettingValues lists the value in force of every setting the shell and the plugins declare.
func (c Config) listSettingValues(target Settings) (settingValues, error) {
	values := settingValues{}
	for _, declared := range settings.GetAppDeclaration() {
		path := append([]string{settings.OwnerApp}, strings.Split(declared.Key, ".")...)
		value, found := target.GetValue(path...)
		if !found {
			continue
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		values[joinSettingPath(path)] = string(encoded)
	}
	for _, plugin := range c.Plugins()() {
		for _, declared := range plugin.Settings {
			path := []string{plugin.ID, declared.Key}
			if value, found := target.GetValue(path...); found {
				encoded, err := json.Marshal(value)
				if err != nil {
					return nil, err
				}
				values[joinSettingPath(path)] = string(encoded)
			} else if declared.DefaultJSON != "" {
				values[joinSettingPath(path)] = declared.DefaultJSON
			}
		}
	}
	return values, nil
}

// watchSettingsFile calls changed after the settings file is saved by anyone, until the returned
// function is called. The folder is watched and not the file: a save replaces the file. The files
// beside it, such as the lock, are no event.
func (c Config) watchSettingsFile(changed func()) func() {
	path, err := c.settingsFile()
	if err != nil {
		c.handleError(err)
		return func() {}
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		c.handleError(err)
		return func() {}
	}
	raw := make(chan notify.EventInfo, settingEventBacklog)
	stop, err := c.getResources().watches.Watch(dir, false, raw)
	if err != nil {
		c.handleError(err)
		return func() {}
	}
	quit := make(chan struct{})
	go debounceSettingFile(filepath.Base(path), raw, quit, changed)
	return func() {
		stop()
		close(quit)
	}
}

// debounceSettingFile calls changed once for the events of one save of the file named. The delay
// runs from the first event, so a file written continuously is still reported.
func debounceSettingFile(name string, raw <-chan notify.EventInfo, quit <-chan struct{}, changed func()) {
	var due <-chan time.Time
	for {
		select {
		case <-quit:
			return
		case event := <-raw:
			if filepath.Base(event.Path()) == name && due == nil {
				due = time.After(settingSaveDelay)
			}
		case <-due:
			due = nil
			changed()
		}
	}
}

// subscribeHost makes the plugin host a reader of the events, once: it takes the settings of its
// plugins and the performance profile through them.
func (c Config) subscribeHost() {
	events := c.getResources().settingEvents
	if events == nil || c.PluginHost == nil {
		return
	}
	events.hostOnce.Do(func() {
		changes, _ := events.subscribe(c)
		go func() {
			for batch := range changes {
				c.tellHost(batch)
			}
		}()
	})
}

// tellHost gives the plugin host the settings of the plugins among those that changed, and the
// performance profile.
func (c Config) tellHost(batch []settingsport.Setting) {
	notifyPluginSettings(c.PluginHost, batch, c.handleError)
	for _, one := range batch {
		if !slices.Equal(one.Path, profilePath()) {
			continue
		}
		var profile string
		if err := json.Unmarshal([]byte(one.JSON), &profile); err != nil {
			c.handleError(err)
			continue
		}
		c.PluginHost.SetPerformanceProfile(context.Background(), hardware.ResolveCompute(profile, c.Hardware).Profile)
	}
}

// profilePath is the path of the performance profile in the settings file.
func profilePath() []string { return []string{settings.OwnerApp, "performance", "profile"} }
