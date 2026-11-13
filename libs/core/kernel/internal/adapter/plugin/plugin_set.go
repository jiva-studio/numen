package plugin

import (
	"errors"
	"fmt"
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

const settingEnabled = "enabled"

// builtInPlugin is a plugin linked into the host's binary: the plugin its manifest describes and
// what makes the handler that serves its backend from the variables a launched process of it is
// given.
type builtInPlugin struct {
	plugin manifest.Discovered
	serve  func(vars map[string]string) (http.Handler, error)
}

// PluginSet is the plugins the host knows, which of them came from the disk or are linked into the
// binary, and the problems the last scan met. It is the one owner of that state.
// modeReader is what the set reads of the mode of the host.
type modeReader interface {
	IsSafeMode() bool
	GetVaultPath() string
}

type PluginSet struct {
	sups   *supervision.Supervisors
	values *SettingValues
	mode   modeReader

	mu       sync.RWMutex
	plugins  map[string]manifest.Discovered
	onDisk   map[string]bool
	builtIns map[string]builtInPlugin
	problems []manifestjson.DiscoveryProblem
}

// PluginSetDeps are what a PluginSet reads: the supervisors of the running plugins, the values of their settings and
// the mode of the host.
type PluginSetDeps struct {
	Supervisors *supervision.Supervisors
	Values      *SettingValues
	Mode        *Mode
}

func NewPluginSet(d PluginSetDeps) *PluginSet {
	return &PluginSet{
		sups: d.Supervisors, values: d.Values, mode: d.Mode,
		plugins:  make(map[string]manifest.Discovered),
		onDisk:   make(map[string]bool),
		builtIns: make(map[string]builtInPlugin),
	}
}

// setView reads the set. The set holds its lock for reading until the function view returned is called.
type setView struct{ set *PluginSet }

func (s *PluginSet) view() (setView, func()) {
	s.mu.RLock()
	return setView{set: s}, s.mu.RUnlock
}

func (v setView) get(pluginID string) (manifest.Discovered, bool) {
	p, ok := v.set.plugins[pluginID]
	return p, ok
}

func (v setView) ordered() []manifest.Discovered {
	result := make([]manifest.Discovered, 0, len(v.set.plugins))
	for _, p := range v.set.plugins {
		result = append(result, p)
	}
	slices.SortFunc(result, func(a, b manifest.Discovered) int { return strings.Compare(a.Manifest.ID, b.Manifest.ID) })
	return result
}

func (v setView) isEnabled(pluginID string) bool {
	settings := v.set.values.get(pluginID)
	p, isKnown := v.set.plugins[pluginID]
	if isKnown {
		settings = p.Manifest.ResolveSettings(settings)
	}
	if val, ok := settings[settingEnabled].(bool); ok {
		return val
	}
	return !p.Manifest.IsOffByDefault
}

func (v setView) settings(pluginID string) map[string]any {
	result := v.set.values.get(pluginID)
	p, ok := v.set.plugins[pluginID]
	if ok {
		result = p.Manifest.ResolveSettings(result)
	}
	if result == nil {
		result = map[string]any{}
	}
	return result
}

func (v setView) requirements(p manifest.Discovered) []manifest.DependencyRequirement {
	return p.Manifest.GetDependencyRequirements(p.Dir, v.settings(p.Manifest.ID))
}

func (v setView) enabled() []manifest.Discovered {
	if v.set.mode.IsSafeMode() {
		return nil
	}
	return slices.DeleteFunc(v.ordered(), func(p manifest.Discovered) bool { return !v.isEnabled(p.Manifest.ID) })
}

func (v setView) hasBackend(p manifest.Discovered) bool {
	_, isBuiltIn := v.set.builtIns[p.Manifest.ID]
	return isBuiltIn || p.Manifest.HasBackend()
}

func (s *PluginSet) getKnownPlugin(pluginID string) (manifest.Discovered, bool) {
	v, release := s.view()
	defer release()
	return v.get(pluginID)
}

func (s *PluginSet) getName(pluginID string) string {
	p, _ := s.getKnownPlugin(pluginID)
	return p.Manifest.Name
}

func (s *PluginSet) isRegistered(pluginID string) bool {
	_, ok := s.getKnownPlugin(pluginID)
	return ok
}

func (s *PluginSet) getPluginSettings(pluginID string) map[string]any {
	v, release := s.view()
	defer release()
	return v.settings(pluginID)
}

func (s *PluginSet) getRequirementIDs(pluginID string) []string {
	v, release := s.view()
	defer release()
	p, ok := v.get(pluginID)
	if !ok {
		return nil
	}
	var ids []string
	for _, req := range v.requirements(p) {
		ids = append(ids, req.ID)
	}
	return ids
}

func (s *PluginSet) listEnabledPlugins() []manifest.Discovered {
	v, release := s.view()
	defer release()
	return v.enabled()
}

func (s *PluginSet) listIDs() []string {
	v, release := s.view()
	defer release()
	ids := make([]string, 0, len(s.plugins))
	for _, p := range v.ordered() {
		ids = append(ids, p.Manifest.ID)
	}
	return ids
}

func (s *PluginSet) isOnDisk(pluginID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.onDisk[pluginID]
}

func (s *PluginSet) getLaunch(p manifest.Discovered) supervision.Launch {
	v, release := s.view()
	defer release()
	launch := supervision.Launch{Plugin: p, BuiltIn: nil, Requirements: v.requirements(p), VaultPath: s.mode.GetVaultPath()}
	if builtIn, ok := s.builtIns[p.Manifest.ID]; ok && !s.onDisk[p.Manifest.ID] {
		launch.BuiltIn = &supervision.BuiltIn{Serve: builtIn.serve}
	}
	return launch
}

func (s *PluginSet) refuseConflicts(discovered []manifest.Discovered, problems []manifestjson.DiscoveryProblem) ([]manifest.Discovered, []manifestjson.DiscoveryProblem) {
	discovered, problems = s.refuseBuiltInProviders(discovered, problems)
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

func (s *PluginSet) replaceDiscovered(discovered []manifest.Discovered, problems []manifestjson.DiscoveryProblem) []string {
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

func (s *PluginSet) removeDiskPluginLocked(pluginID string) {
	delete(s.onDisk, pluginID)
	if builtIn, ok := s.builtIns[pluginID]; ok {
		s.plugins[pluginID] = builtIn.plugin
		s.sups.Register(builtIn.plugin)
		return
	}
	delete(s.plugins, pluginID)
}

func (s *PluginSet) isCoveredByProblemLocked(pluginID string) bool {
	dir := s.plugins[pluginID].Dir
	for _, problem := range s.problems {
		if problem.Dir == dir || problem.Dir == filepath.Dir(dir) {
			return true
		}
	}
	return false
}

func (s *PluginSet) put(p manifest.Discovered, sup *lifecycle.Supervisor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.plugins[p.Manifest.ID] = p
	s.sups.Put(p.Manifest.ID, sup)
}

func (s *PluginSet) addBuiltIn(p manifest.Discovered, serve func(vars map[string]string) (http.Handler, error)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := p.Manifest.ID
	if _, exists := s.builtIns[id]; exists {
		return errBuiltInRegistered
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
	s.builtIns[id] = builtInPlugin{plugin: p, serve: serve}
	if _, registered := s.plugins[id]; !registered {
		s.plugins[id] = p
		s.sups.Register(p)
	}
	return nil
}

func (v setView) listProblems() []manifestjson.DiscoveryProblem {
	return v.set.problems
}

// GetVersion returns the version of the manifest of the plugin, or none for a plugin the set does not know.
func (s *PluginSet) GetVersion(pluginID string) string {
	p, _ := s.getKnownPlugin(pluginID)
	return p.Manifest.Version
}

// errProviderIDTaken is a plugin found on disk under the id of a built-in plugin that provides connection
// types. The keys of a connection go to the plugin that provides its type, so the built-in plugin stays.
var errProviderIDTaken = errors.New("a built-in plugin of this id provides connection types")

func (s *PluginSet) refuseBuiltInProviders(discovered []manifest.Discovered, problems []manifestjson.DiscoveryProblem) ([]manifest.Discovered, []manifestjson.DiscoveryProblem) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	kept := make([]manifest.Discovered, 0, len(discovered))
	for _, p := range discovered {
		if builtIn, isBuiltIn := s.builtIns[p.Manifest.ID]; isBuiltIn && len(builtIn.plugin.Manifest.Contributes.ConnectionTypes) > 0 {
			problems = append(slices.Clone(problems), manifestjson.DiscoveryProblem{
				Dir: p.Dir, Err: fmt.Errorf("%w: rejected the plugin in %s", errProviderIDTaken, p.Dir),
			})
			continue
		}
		kept = append(kept, p)
	}
	return kept, problems
}
