package rowoptions_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/modelcalltest"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/rowoptions"
)

// memoryLists keeps the option lists in memory, as the cache files of the plugins do across processes.
type memoryLists struct {
	mu       sync.Mutex
	byPlugin map[string][]connection.OptionList
}

func (m *memoryLists) ReadOptionLists(pluginID string) ([]connection.OptionList, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	lists, isKept := m.byPlugin[pluginID]
	return slices.Clone(lists), isKept
}

func (m *memoryLists) WriteOptionLists(pluginID string, lists []connection.OptionList) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.byPlugin[pluginID] = slices.Clone(lists)
}

// runningProviders opens the provider of the world while isRunning is set, and counts the openings.
type runningProviders struct {
	mu        sync.Mutex
	provider  *modelcalltest.Provider
	isRunning bool
	opened    int
}

func (r *runningProviders) OpenRunningProvider(context.Context, string) (connectionport.Provider, func(), bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.isRunning {
		return nil, nil, false
	}
	r.opened++
	return r.provider, func() {}, true
}

func (r *runningProviders) setRunning(isRunning bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.isRunning = isRunning
}

func (r *runningProviders) countOpenCalls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.opened
}

// revisions names a revision for each plugin by id; a plugin left out has none.
type revisions struct {
	mu       sync.Mutex
	byPlugin map[string]string
}

func (r *revisions) set(pluginID, revision string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byPlugin[pluginID] = revision
}

func (r *revisions) GetPluginConfigRevision(pluginID string) string {
	revision, _, _ := r.getGrantRevision(pluginID)
	return revision
}

func (r *revisions) getGrantRevision(pluginID string) (string, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	revision, isSet := r.byPlugin[pluginID]
	return revision, isSet, nil
}

func (s *setup) restart() *rowoptions.Options {
	return s.openOver(s.world.Source)
}

func (s *setup) openOver(source connectionport.ProviderSource) *rowoptions.Options {
	return rowoptions.NewOptions(rowoptions.Deps{
		Resolve: s.world.Resolve, ReadSecrets: s.world.ReadSecrets, Source: source, Running: s.running,
		Lists: rowoptions.NewLists(s.files), Configs: s.configs, Grants: s.grants.getGrantRevision,
	})
}

func (s *setup) countCalls() int { return len(s.world.Provider.Calls()) }

func (s *setup) mustListModels(t *testing.T) {
	t.Helper()
	if _, err := s.listModels(t.Context(), modelcalltest.EmbedOperation); err != nil {
		t.Fatal(err)
	}
}

func TestListOptionsAnswersAStoredListWithoutOpeningTheProvider(t *testing.T) {
	s := newSetup(t)
	s.mustListModels(t)
	s.world.Source.FailOpen(modelcalltest.ProviderPluginID, errors.New("the plugin must not start"))

	options, err := s.restart().ListOptions(t.Context(), s.entry.ID, modelcalltest.ModelOption, map[string]string{"reasoning": "low"}, []string{modelcalltest.EmbedOperation})

	if err != nil || len(options) != 2 || options[0].Value != "model-1" {
		t.Errorf("options = %v, %v, want the kept two", options, err)
	}
	if calls, opened := s.countCalls(), s.world.Source.Opened(); calls != 1 || len(opened) != 1 {
		t.Errorf("calls = %d, opened = %v, want the first listing alone", calls, opened)
	}
}

func TestListOptionsAsksTheProviderWhenNoListIsStored(t *testing.T) {
	for name, between := range map[string]func(*testing.T, *setup){
		"for a connection that changed": func(t *testing.T, s *setup) {
			t.Helper()
			s.entry = s.world.Edit(t, s.entry, "First renamed")
		},
		"for another version of its plugin": func(_ *testing.T, s *setup) {
			s.world.SetPluginVersion(modelcalltest.ProviderPluginID, "2.0.0")
		},
	} {
		s := newSetup(t)
		s.mustListModels(t)
		between(t, s)

		s.mustListModels(t)

		if calls := s.countCalls(); calls != 2 {
			t.Errorf("%s: the provider was asked %d times, want twice", name, calls)
		}
	}
}

func TestRefreshOptionsAsksTheProviderAndKeepsTheAnswer(t *testing.T) {
	s := newSetup(t)
	s.mustListModels(t)

	if _, err := s.options.RefreshOptions(t.Context(), s.entry.ID, modelcalltest.ModelOption, map[string]string{"reasoning": "low"}, []string{modelcalltest.EmbedOperation}); err != nil {
		t.Fatal(err)
	}
	s.mustListModels(t)

	if calls := s.countCalls(); calls != 2 {
		t.Errorf("the provider was asked %d times, want twice", calls)
	}
}

func TestRefreshRunningAsksARunningPluginAndKeepsTheAnswerForTheConnectionAsItIsNow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newSetup(t)
		s.mustListModels(t)
		s.entry = s.world.Edit(t, s.entry, "First renamed")
		s.running.setRunning(true)

		s.options.RefreshRunning(s.entry.ID)
		synctest.Wait()
		s.mustListModels(t)

		if opened, calls := s.running.countOpenCalls(), s.countCalls(); opened != 1 || calls != 2 {
			t.Errorf("running opened = %d, calls = %d, want the refresh alone after the first listing", opened, calls)
		}
		if opened := s.world.Source.Opened(); len(opened) != 1 {
			t.Errorf("the source opened %v, want the first listing alone", opened)
		}
	})
}

func TestRefreshRunningDoesNotStartAPluginThatDoesNotRun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newSetup(t)
		s.mustListModels(t)

		s.options.RefreshRunning(s.entry.ID)
		synctest.Wait()

		if opened, calls := s.world.Source.Opened(), s.countCalls(); len(opened) != 1 || calls != 1 {
			t.Errorf("opened = %v, calls = %d, want the first listing alone", opened, calls)
		}
	})
}

func TestAFailedRefreshPreservesTheStoredList(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newSetup(t)
		s.mustListModels(t)
		s.running.setRunning(true)
		s.world.Provider.Fail(address, &connection.TryNextError{Reason: connection.FailureReasonUnavailable})

		s.options.RefreshRunning(s.entry.ID)
		synctest.Wait()
		options, err := s.listModels(t.Context(), modelcalltest.EmbedOperation)

		if err != nil || len(options) != 2 || s.countCalls() != 2 {
			t.Errorf("options = %v, %v, calls = %d, want the kept two after the failed refresh", options, err, s.countCalls())
		}
	})
}

func TestRefreshConnectionAsksTheProviderForTheChangedConnection(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newSetup(t)
		s.mustListModels(t)
		s.entry = s.world.Edit(t, s.entry, "First renamed")

		s.options.RefreshConnection(s.entry.ID)
		synctest.Wait()
		s.mustListModels(t)

		if opened, calls := s.world.Source.Opened(), s.countCalls(); len(opened) != 2 || calls != 2 {
			t.Errorf("opened = %v, calls = %d, want the first listing and the refresh", opened, calls)
		}
	})
}

func TestRefreshOfAConnectionWithNoStoredListAsksNothing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newSetup(t)
		s.running.setRunning(true)

		s.options.RefreshConnection(s.entry.ID)
		s.options.RefreshRunning(s.entry.ID)
		synctest.Wait()

		if calls := s.countCalls(); calls != 0 {
			t.Errorf("the provider was asked %d times, want none", calls)
		}
	})
}

func TestAListingForAnotherValueOfAFieldBeforeAsksTheProvider(t *testing.T) {
	s := newSetup(t)
	for _, model := range []string{"model-1", "model-2"} {
		if _, err := s.listReasoningLevels(t.Context(), model); err != nil {
			t.Fatal(err)
		}
	}

	if calls := s.countCalls(); calls != 2 {
		t.Errorf("the provider was asked %d times, want twice", calls)
	}
}

func TestAStoredListIsAnsweredForAnyValueOfTheFieldAndTheFieldsAfterIt(t *testing.T) {
	s := newSetup(t)
	s.mustListModels(t)

	options, err := s.options.ListOptions(t.Context(), s.entry.ID, modelcalltest.ModelOption,
		map[string]string{modelcalltest.ModelOption: "model-2", reasoningField: "high"}, []string{modelcalltest.EmbedOperation})

	if err != nil || len(options) != 2 || s.countCalls() != 1 {
		t.Errorf("options = %v, %v, calls = %d, want the kept two with no second ask", options, err, s.countCalls())
	}
}
