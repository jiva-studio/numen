package providerbridge

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const (
	// DetectLimit is how long a provider has to start and answer connections.detected.
	DetectLimit = 30 * time.Second
	// RefreshInterval is how long Refresh leaves a plugin alone after it was last asked.
	RefreshInterval = 10 * time.Minute
)

// answer is what the provider of a type reported, and the version of its plugin then.
type answer struct {
	pluginID      string
	pluginVersion string
	connections   []connection.Connection
}

// Detection keeps the connections the providers report ready on this machine, by type. A type takes part
// when it lists connection.OperationDetected among its operations; until its provider answers, it lists
// none. An answer counts only for the version of the plugin that gave it. A detected connection is not
// stored with the connections: its id is the type id, a slash and its name. Where files are set, the
// answers of each plugin are kept in them, so the providers are not asked again after a restart.
type Detection struct {
	types  connectionport.Types
	source connectionport.ProviderSource
	clock  machineport.Clock

	mu       sync.Mutex
	answers  map[string]answer
	askedAt  map[string]time.Time
	running  map[string]chan struct{}
	files    DetectionFiles
	loaded   map[string]bool
	onChange func()
	report   func(error)

	writeMu sync.Mutex
}

// NewDetection returns the detection over the registry of types and the providers of the plugins, with no
// answer yet. clock times the refreshes.
func NewDetection(types connectionport.Types, source connectionport.ProviderSource, clock machineport.Clock) *Detection {
	return &Detection{
		types: types, source: source, clock: clock, answers: map[string]answer{}, askedAt: map[string]time.Time{},
		running: map[string]chan struct{}{}, loaded: map[string]bool{}, onChange: func() {}, report: func(error) {},
	}
}

// SetFiles sets the files the answers are kept in; the answers of a plugin are read from them on its first
// use after.
func (d *Detection) SetFiles(files DetectionFiles) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.files, d.loaded = files, map[string]bool{}
}

// Follow sets what runs after an answer changed the detected connections, and what receives the failures of
// a provider and of the store file; it replaces what was set before.
func (d *Detection) Follow(onChange func(), report func(error)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.onChange, d.report = onChange, report
}

// ListDetected returns the connections the providers last reported for the available types, in the order of
// the types and then of the answers. It does not wait for a provider.
func (d *Detection) ListDetected() []connection.Connection {
	types := d.types.ListTypes()
	d.mu.Lock()
	defer d.mu.Unlock()
	var list []connection.Connection
	for _, one := range types {
		if !isDetecting(one) {
			continue
		}
		d.load(one.PluginID)
		if found, isAnswered := d.getAnswer(one); isAnswered {
			list = append(list, found...)
		}
	}
	return list
}

// IsPending reports whether the type is available, serves detection and its provider has not answered yet
// in the version of its plugin installed now.
func (d *Detection) IsPending(typeID string) bool {
	found, isFound := d.types.GetType(typeID)
	if !isFound || !isDetecting(found) {
		return false
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.load(found.PluginID)
	_, isAnswered := d.getAnswer(found)
	return !isAnswered
}

// Detect asks the provider of each available type of the plugin that serves detection, which starts the
// plugin, and waits for the answers. A detection of the plugin that is running already is waited for. A
// provider that fails keeps the answer it gave before. Of an answer, one connection of each valid name, ignoring
// case, is kept, with the values of the fields of the type that are neither secret nor fixed.
func (d *Detection) Detect(ctx context.Context, pluginID string) {
	done, isMine := d.begin(pluginID)
	if !isMine {
		select {
		case <-done:
		case <-ctx.Done():
		}
		return
	}
	defer d.end(pluginID, done)
	onChange, report := d.getFollowers()
	types := d.listDetectingTypes(pluginID)
	if len(types) == 0 {
		return
	}
	provider, release, err := d.source.OpenProvider(ctx, pluginID)
	if err != nil {
		report(fmt.Errorf("detect the connections of %s: %w", pluginID, err))
		return
	}
	defer release()
	isChanged := false
	for _, one := range types {
		found, err := provider.ListDetected(ctx, modelcall.ProviderCall{
			Type: one.Declaration.ID, Fields: map[string]string{}, Secrets: map[string]string{}, Options: map[string]string{},
		})
		if err != nil {
			report(fmt.Errorf("detect the connections of %s: %w", one.GetID(), err))
			continue
		}
		isChanged = d.store(one, newDetectedConnections(one, found)) || isChanged
	}
	if isChanged {
		if err := d.writeStoreFile(pluginID); err != nil {
			report(fmt.Errorf("keep the detected connections: %w", err))
		}
		onChange()
	}
}

// Refresh asks the provider of every plugin with a type that serves detection again, each in the background
// and within DetectLimit. A plugin whose detection is running, or that was asked within RefreshInterval, is
// not asked.
func (d *Detection) Refresh() {
	for _, pluginID := range d.listDetectingPlugins() {
		d.mu.Lock()
		_, isRunning := d.running[pluginID]
		askedAt, isAsked := d.askedAt[pluginID]
		isRecent := isAsked && d.clock.Now().Sub(askedAt) < RefreshInterval
		d.mu.Unlock()
		if isRunning || isRecent {
			continue
		}
		d.DetectInBackground(pluginID)
	}
}

// DetectInBackground runs Detect for the plugin on a goroutine of its own, within DetectLimit.
func (d *Detection) DetectInBackground(pluginID string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), DetectLimit)
		defer cancel()
		d.Detect(ctx, pluginID)
	}()
}

func isDetecting(one connection.Type) bool {
	return one.IsAvailable && one.Declaration.HasOperation(connection.OperationDetected)
}

func (d *Detection) begin(pluginID string) (chan struct{}, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if done, isRunning := d.running[pluginID]; isRunning {
		return done, false
	}
	done := make(chan struct{})
	d.running[pluginID] = done
	d.askedAt[pluginID] = d.clock.Now()
	return done, true
}

func (d *Detection) end(pluginID string, done chan struct{}) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.running, pluginID)
	close(done)
}

func (d *Detection) getFollowers() (onChange func(), report func(error)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.onChange, d.report
}

func (d *Detection) listDetectingTypes(pluginID string) []connection.Type {
	var types []connection.Type
	for _, one := range d.types.ListTypes() {
		if one.PluginID == pluginID && isDetecting(one) {
			types = append(types, one)
		}
	}
	return types
}

func (d *Detection) listDetectingPlugins() []string {
	var plugins []string
	for _, one := range d.types.ListTypes() {
		if isDetecting(one) && !slices.Contains(plugins, one.PluginID) {
			plugins = append(plugins, one.PluginID)
		}
	}
	return plugins
}

func (d *Detection) getAnswer(declared connection.Type) ([]connection.Connection, bool) {
	found, isAnswered := d.answers[declared.GetID()]
	if !isAnswered || found.pluginVersion != declared.PluginVersion {
		return nil, false
	}
	return found.connections, true
}

func (d *Detection) store(declared connection.Type, found []connection.Connection) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.load(declared.PluginID)
	before, isAnswered := d.getAnswer(declared)
	d.answers[declared.GetID()] = answer{pluginID: declared.PluginID, pluginVersion: declared.PluginVersion, connections: found}
	return !isAnswered || !reflect.DeepEqual(before, found)
}

func newDetectedConnections(declared connection.Type, found []modelcall.DetectedConnection) []connection.Connection {
	list := []connection.Connection{}
	for _, one := range found {
		if !connection.IsValidName(one.Name) || slices.ContainsFunc(list, func(kept connection.Connection) bool { return strings.EqualFold(kept.Name, one.Name) }) {
			continue
		}
		fields := map[string]any{}
		for _, field := range declared.Declaration.Fields {
			value, isSet := one.Fields[field.Key]
			if isSet && field.Type != manifest.ConnectionFieldSecret && field.FixedValue == nil {
				fields[field.Key] = readFieldValue(field, value)
			}
		}
		list = append(list, connection.Connection{
			ID: connection.GetDetectedID(declared.GetID(), one.Name), TypeID: declared.GetID(), Name: one.Name, Fields: fields,
			Path: one.Path,
		})
	}
	return list
}

func readFieldValue(field manifest.ConnectionField, text string) any {
	switch field.Type {
	case "number":
		if number, err := strconv.ParseFloat(strings.TrimSpace(text), 64); err == nil {
			return number
		}
	case "boolean":
		if isOn, err := strconv.ParseBool(strings.TrimSpace(text)); err == nil {
			return isOn
		}
	}
	return text
}
