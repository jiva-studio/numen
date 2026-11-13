package rowoptions_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/modelcalltest"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/rowoptions"
)

func (s *setup) countCalls() int { return len(s.world.Provider.Calls()) }

func (s *setup) listModelsTwice(t *testing.T, between func()) int {
	t.Helper()
	if _, err := s.listModels(t.Context(), modelcalltest.EmbedOperation); err != nil {
		t.Fatal(err)
	}
	between()
	if _, err := s.listModels(t.Context(), modelcalltest.EmbedOperation); err != nil {
		t.Fatal(err)
	}
	return s.countCalls()
}

func TestASecondListingWithinTheCacheTTLDoesNotOpenTheProvider(t *testing.T) {
	s := newSetup(t)

	calls := s.listModelsTwice(t, func() { s.world.Clock.Advance(rowoptions.CacheTTL - time.Second) })

	if calls != 1 || len(s.world.Source.Opened()) != 1 {
		t.Errorf("calls = %d, opened = %v, want one of each", calls, s.world.Source.Opened())
	}
}

func TestAListingAsksTheProviderAgain(t *testing.T) {
	for name, between := range map[string]func(*testing.T, *setup){
		"after the cache TTL": func(_ *testing.T, s *setup) { s.world.Clock.Advance(rowoptions.CacheTTL) },
		"after a change of the connection": func(t *testing.T, s *setup) {
			t.Helper()
			s.entry = s.world.Edit(t, s.entry, "First renamed")
		},
		"after a change of the version of its plugin": func(_ *testing.T, s *setup) {
			s.world.SetPluginVersion(modelcalltest.ProviderPluginID, "2.0.0")
		},
	} {
		s := newSetup(t)

		calls := s.listModelsTwice(t, func() { between(t, s) })

		if calls != 2 {
			t.Errorf("%s: the provider was asked %d times, want twice", name, calls)
		}
	}
}

func TestAListingForOtherRowOptionsAsksTheProvider(t *testing.T) {
	s := newSetup(t)
	if _, err := s.listModels(t.Context(), modelcalltest.EmbedOperation); err != nil {
		t.Fatal(err)
	}

	if _, err := s.options.ListOptions(t.Context(), s.entry.ID, modelcalltest.ModelOption, map[string]string{"reasoning": "high"}, []string{modelcalltest.EmbedOperation}); err != nil {
		t.Fatal(err)
	}

	if calls := s.countCalls(); calls != 2 {
		t.Errorf("the provider was asked %d times, want twice", calls)
	}
}
