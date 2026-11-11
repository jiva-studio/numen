// Package pluginset holds the plugins the host knows and the values the user stored for their settings. Every part of
// the host that answers for a plugin reads them through it.
package pluginset

import (
	"errors"
	"net/http"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

// SettingEnabled is the key of the setting that switches a plugin on or off.
const SettingEnabled = "enabled"

var (
	// ErrNotFound is what a call fails with for a plugin the host does not know.
	ErrNotFound = errors.New("plugin not found")
	// ErrBuiltInRegistered is what the registration of a built-in plugin fails with when one of its id is registered.
	ErrBuiltInRegistered = errors.New("a built-in plugin of this id is registered")
)

// modeState is what the set reads of the mode of the host.
type modeState interface {
	IsSafeMode() bool
	GetVaultPath() string
}

// linkedPlugin is a plugin linked into the host's binary: the plugin its manifest describes and
// what makes the handler that serves its backend from the variables a launched process of it is
// given.
type linkedPlugin struct {
	plugin manifest.Discovered
	serve  func(vars map[string]string) (http.Handler, error)
}

// Set is the plugins the host knows, which of them came from the disk or are linked into the
// binary, and the problems the last scan met. It is the one owner of that state.
type Set struct {
	sups   *supervision.Supervisors
	values *Values
	mode   modeState

	scan     ScanLock
	mu       sync.RWMutex
	plugins  map[string]manifest.Discovered
	onDisk   map[string]bool
	builtIns map[string]linkedPlugin
	problems []manifestjson.DiscoveryProblem
}

// New returns an empty set over the supervisors, the stored values and the mode of the host.
func New(sups *supervision.Supervisors, values *Values, mode modeState) *Set {
	return &Set{
		sups: sups, values: values, mode: mode,
		plugins:  make(map[string]manifest.Discovered),
		onDisk:   make(map[string]bool),
		builtIns: make(map[string]linkedPlugin),
	}
}

// View reads the set. The set holds its lock for reading until the function view returned is called.
type View struct{ set *Set }

func (s *Set) View() (View, func()) {
	s.mu.RLock()
	return View{set: s}, s.mu.RUnlock
}

func (v View) Get(pluginID string) (manifest.Discovered, bool) {
	p, ok := v.set.plugins[pluginID]
	return p, ok
}

func (v View) Ordered() []manifest.Discovered {
	result := make([]manifest.Discovered, 0, len(v.set.plugins))
	for _, p := range v.set.plugins {
		result = append(result, p)
	}
	slices.SortFunc(result, func(a, b manifest.Discovered) int { return strings.Compare(a.Manifest.ID, b.Manifest.ID) })
	return result
}

func (v View) IsEnabled(pluginID string) bool {
	settings := v.set.values.Get(pluginID)
	if p, ok := v.set.plugins[pluginID]; ok {
		settings = p.Manifest.ResolveSettings(settings)
	}
	if val, ok := settings[SettingEnabled].(bool); ok {
		return val
	}
	return true
}

func (v View) Settings(pluginID string) map[string]any {
	result := v.set.values.Get(pluginID)
	p, ok := v.set.plugins[pluginID]
	if ok {
		result = p.Manifest.ResolveSettings(result)
	}
	if result == nil {
		result = map[string]any{}
	}
	if ok {
		for _, name := range p.Manifest.GetSecretNames() {
			delete(result, name)
		}
	}
	return result
}

func (v View) Requirements(p manifest.Discovered) []manifest.DependencyRequirement {
	return p.Manifest.GetDependencyRequirements(p.Dir, v.Settings(p.Manifest.ID))
}

func (v View) Enabled() []manifest.Discovered {
	if v.set.mode.IsSafeMode() {
		return nil
	}
	return slices.DeleteFunc(v.Ordered(), func(p manifest.Discovered) bool { return !v.IsEnabled(p.Manifest.ID) })
}

func (v View) HasBackend(p manifest.Discovered) bool {
	_, isBuiltIn := v.set.builtIns[p.Manifest.ID]
	return isBuiltIn || p.Manifest.HasBackend()
}

func (s *Set) GetKnown(pluginID string) (manifest.Discovered, bool) {
	v, release := s.View()
	defer release()
	return v.Get(pluginID)
}

func (s *Set) GetName(pluginID string) string {
	p, _ := s.GetKnown(pluginID)
	return p.Manifest.Name
}

func (s *Set) IsRegistered(pluginID string) bool {
	_, ok := s.GetKnown(pluginID)
	return ok
}

func (s *Set) GetPluginSettings(pluginID string) map[string]any {
	v, release := s.View()
	defer release()
	return v.Settings(pluginID)
}

func (s *Set) GetRequirementIDs(pluginID string) []string {
	v, release := s.View()
	defer release()
	p, ok := v.Get(pluginID)
	if !ok {
		return nil
	}
	var ids []string
	for _, req := range v.Requirements(p) {
		ids = append(ids, req.ID)
	}
	return ids
}

func (s *Set) ListEnabled() []manifest.Discovered {
	v, release := s.View()
	defer release()
	return v.Enabled()
}

func (s *Set) ListIDs() []string {
	v, release := s.View()
	defer release()
	ids := make([]string, 0, len(s.plugins))
	for _, p := range v.Ordered() {
		ids = append(ids, p.Manifest.ID)
	}
	return ids
}

func (s *Set) HasProblems() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.problems) > 0
}

func (s *Set) IsOnDisk(pluginID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.onDisk[pluginID]
}

func (s *Set) GetLaunch(p manifest.Discovered) supervision.Launch {
	v, release := s.View()
	defer release()
	launch := supervision.Launch{Plugin: p, BuiltIn: nil, Requirements: v.Requirements(p), VaultPath: s.mode.GetVaultPath()}
	if builtIn, ok := s.builtIns[p.Manifest.ID]; ok && !s.onDisk[p.Manifest.ID] {
		launch.BuiltIn = &supervision.BuiltIn{Serve: builtIn.serve}
	}
	return launch
}

func (s *Set) RefuseConflicts(discovered []manifest.Discovered, problems []manifestjson.DiscoveryProblem) ([]manifest.Discovered, []manifestjson.DiscoveryProblem) {
	s.mu.RLock()
	ordered := make([]manifest.Discovered, 0, len(s.builtIns)+len(discovered))
	for id, builtIn := range s.builtIns {
		if !slices.ContainsFunc(discovered, func(p manifest.Discovered) bool { return p.Manifest.ID == id }) {
			ordered = append(ordered, builtIn.plugin)
		}
	}
	s.mu.RUnlock()
	byID := func(a, b manifest.Discovered) int { return strings.Compare(a.Manifest.ID, b.Manifest.ID) }
	slices.SortFunc(ordered, byID)
	sorted := slices.Clone(discovered)
	slices.SortFunc(sorted, byID)
	// Plugins linked into the binary come first, then the discovered ones, each group in the order of the
	// ids: the first plugin to declare a tab kind, a reader or a type of dropped data keeps it.
	conflicts := manifest.ListContributionConflicts(append(ordered, sorted...))
	if len(conflicts) == 0 {
		return discovered, problems
	}
	conflicting := make(map[string]bool, len(conflicts))
	for _, conflict := range conflicts {
		conflicting[conflict.PluginID] = true
	}
	accepted := make([]manifest.Discovered, 0, len(discovered))
	for _, p := range discovered {
		if !conflicting[p.Manifest.ID] {
			accepted = append(accepted, p)
		}
	}
	problems = slices.Clone(problems)
	for _, conflict := range conflicts {
		for _, p := range sorted {
			if p.Manifest.ID == conflict.PluginID {
				problems = append(problems, manifestjson.DiscoveryProblem{Dir: p.Dir, Err: conflict})
			}
		}
	}
	return accepted, problems
}

func (s *Set) ReplaceDiscovered(discovered []manifest.Discovered, problems []manifestjson.DiscoveryProblem) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.problems = problems
	present := make(map[string]bool, len(discovered))
	var stale []string
	for _, p := range discovered {
		id := p.Manifest.ID
		present[id] = true
		if recognized, ok := s.plugins[id]; ok && !reflect.DeepEqual(recognized, p) {
			stale = append(stale, id)
		}
		s.plugins[id] = p
		s.onDisk[id] = true
		s.sups.Register(p)
	}
	for id := range s.onDisk {
		if !present[id] && !s.isCoveredByProblemLocked(id) {
			s.removeDiskPluginLocked(id)
			stale = append(stale, id)
		}
	}
	return stale
}

func (s *Set) removeDiskPluginLocked(pluginID string) {
	delete(s.onDisk, pluginID)
	if builtIn, ok := s.builtIns[pluginID]; ok {
		s.plugins[pluginID] = builtIn.plugin
		s.sups.Register(builtIn.plugin)
		return
	}
	delete(s.plugins, pluginID)
}

func (s *Set) isCoveredByProblemLocked(pluginID string) bool {
	dir := s.plugins[pluginID].Dir
	for _, problem := range s.problems {
		if problem.Dir == dir || problem.Dir == filepath.Dir(dir) {
			return true
		}
	}
	return false
}

func (s *Set) Put(p manifest.Discovered, sup *lifecycle.Supervisor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.plugins[p.Manifest.ID] = p
	s.sups.Put(p.Manifest.ID, sup)
}

func (s *Set) AddBuiltIn(p manifest.Discovered, serve func(vars map[string]string) (http.Handler, error)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := p.Manifest.ID
	if _, exists := s.builtIns[id]; exists {
		return ErrBuiltInRegistered
	}
	others := make([]manifest.Discovered, 0, len(s.plugins))
	for otherID, other := range s.plugins {
		if otherID != id {
			others = append(others, other)
		}
	}
	slices.SortFunc(others, func(a, b manifest.Discovered) int { return strings.Compare(a.Manifest.ID, b.Manifest.ID) })
	if conflicts := manifest.ListContributionConflicts(append(others, p)); len(conflicts) > 0 {
		return conflicts[len(conflicts)-1]
	}
	s.builtIns[id] = linkedPlugin{plugin: p, serve: serve}
	if _, registered := s.plugins[id]; !registered {
		s.plugins[id] = p
		s.sups.Register(p)
	}
	return nil
}

// ListProblems returns the problems the last scan met.
func (v View) ListProblems() []manifestjson.DiscoveryProblem {
	return v.set.problems
}

// Scan returns the lock that keeps a scan of the plugin folders apart from the start of a plugin.
func (s *Set) Scan() *ScanLock {
	return &s.scan
}
