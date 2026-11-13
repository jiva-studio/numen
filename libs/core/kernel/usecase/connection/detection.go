package connection

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
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// DetectLimit is how long a provider has to start and answer connections.detected.
const DetectLimit = 30 * time.Second

// Detection keeps the connections the providers report ready on this machine, by type. A type takes part
// when it lists connection.OperationDetected among its operations; until its provider answers, it lists
// none. A detected connection is not stored: its id is the type id, a slash and its name.
type Detection struct {
	types  connectionport.Types
	source connectionport.ProviderSource

	mu       sync.Mutex
	answers  map[string][]connection.Connection
	running  map[string]chan struct{}
	onChange func()
	report   func(error)
}

var _ connectionport.Detected = (*Detection)(nil)

// NewDetection returns the detection over the registry of types and the providers of the plugins, with no
// answer yet.
func NewDetection(types connectionport.Types, source connectionport.ProviderSource) *Detection {
	return &Detection{
		types: types, source: source, answers: map[string][]connection.Connection{}, running: map[string]chan struct{}{},
		onChange: func() {}, report: func(error) {},
	}
}

// Follow sets what runs after an answer changed the detected connections, and what receives the failures of
// a provider; it replaces what was set before.
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
		if isDetecting(one) {
			list = append(list, d.answers[one.GetID()]...)
		}
	}
	return list
}

// IsPending reports whether the type is available, serves detection and its provider has not answered yet.
func (d *Detection) IsPending(typeID string) bool {
	found, isFound := d.types.GetType(typeID)
	if !isFound || !isDetecting(found) {
		return false
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	_, isAnswered := d.answers[typeID]
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
		isChanged = d.store(one.GetID(), newDetectedConnections(one, found)) || isChanged
	}
	if isChanged {
		onChange()
	}
}

// Refresh asks the provider of every plugin with a type that serves detection again, each in the background
// and within DetectLimit. A plugin whose detection is running is not asked twice.
func (d *Detection) Refresh() {
	for _, pluginID := range d.listDetectingPlugins() {
		d.mu.Lock()
		_, isRunning := d.running[pluginID]
		d.mu.Unlock()
		if isRunning {
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

func (d *Detection) store(typeID string, found []connection.Connection) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	before, isAnswered := d.answers[typeID]
	d.answers[typeID] = found
	return !isAnswered || !reflect.DeepEqual(before, found)
}

func newDetectedConnections(declared connection.Type, found []modelcall.DetectedConnection) []connection.Connection {
	list := []connection.Connection{}
	for _, one := range found {
		if !connection.IsValidName(one.Name) || isNameTaken(list, one.Name, "") {
			continue
		}
		fields := map[string]any{}
		for _, field := range declared.Declaration.Fields {
			value, isSet := one.Fields[field.Key]
			if isSet && field.Type != secretFieldType && field.FixedValue == nil {
				fields[field.Key] = readFieldValue(field, value)
			}
		}
		list = append(list, connection.Connection{
			ID: connection.GetDetectedID(declared.GetID(), one.Name), TypeID: declared.GetID(), Name: one.Name, Fields: fields,
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

// noDetection is the detection of an installation whose types serve no detection.
type noDetection struct{}

func (noDetection) ListDetected() []connection.Connection { return nil }

func (noDetection) IsPending(string) bool { return false }

func (noDetection) Detect(context.Context, string) {}

func (noDetection) Refresh() {}
