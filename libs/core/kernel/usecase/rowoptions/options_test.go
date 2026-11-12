package modellist_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	modelcalldomain "github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/modelcalltest"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/modellist"
)

const address = "https://one.test/v1"

type setup struct {
	world *modelcalltest.World
	lists *modellist.Lists
	entry modelcalltest.Entry
}

func newSetup(t *testing.T) *setup {
	t.Helper()
	w := modelcalltest.New()
	return &setup{
		world: w, entry: w.AddServer(t, "First", address, "sk-one"),
		lists: modellist.NewLists(w.Resolve, w.ReadSecrets, w.Source),
	}
}

func TestListModelsReturnsTheModelsTheProviderLabelsForTheOperation(t *testing.T) {
	s := newSetup(t)

	models, err := s.lists.ListModels(t.Context(), s.entry.ID, "embed")

	want := []modelcalldomain.Model{{ID: "model-1", Title: "Model one", Operations: []string{"embed"}}, {ID: "model-2"}}
	if err != nil || !slices.EqualFunc(models, want, func(a, b modelcalldomain.Model) bool {
		return a.ID == b.ID && a.Title == b.Title && slices.Equal(a.Operations, b.Operations)
	}) {
		t.Errorf("models = %v, %v, want %v", models, err, want)
	}
}

func TestListModelsCallsTheProviderWithTheConnectionAndItsKeyAndNoModel(t *testing.T) {
	s := newSetup(t)

	if _, err := s.lists.ListModels(t.Context(), s.entry.ID, "complete"); err != nil {
		t.Fatal(err)
	}

	got := s.world.Provider.Calls()[0]
	if got.Operation != "list models" || got.Call.Model != "" || got.Call.Secrets["apiKey"] != "sk-one" || got.Call.Fields["baseUrl"] != address {
		t.Errorf("call = %+v", got)
	}
	if providers := s.world.Source.Opened(); !slices.Equal(providers, []string{modelcalltest.ProviderPluginID}) || s.world.Source.Released() != 1 {
		t.Errorf("opened = %v, released = %d", providers, s.world.Source.Released())
	}
}

func TestListReasoningLevelsAsksTheProviderAboutTheModel(t *testing.T) {
	s := newSetup(t)

	levels, err := s.lists.ListReasoningLevels(t.Context(), s.entry.ID, "model-1")

	if err != nil || len(levels) != 2 || levels[0] != (modelcalldomain.ReasoningLevel{Value: "low", Title: "Low"}) {
		t.Fatalf("levels = %v, %v", levels, err)
	}
	if got := s.world.Provider.Calls()[0]; got.Operation != "list reasoning levels" || got.Call.Model != "model-1" {
		t.Errorf("call = %+v", got)
	}
}

func TestAConnectionThatIsNotInTheListIsNotFound(t *testing.T) {
	s := newSetup(t)
	if _, err := s.lists.ListModels(t.Context(), "c0000dead", "embed"); !errors.Is(err, connection.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestAConnectionWhoseTypeIsNotProvidedHasNoModels(t *testing.T) {
	s := newSetup(t)
	s.world.SetAvailable(modelcalltest.ProviderPluginID, false)
	if _, err := s.lists.ListModels(t.Context(), s.entry.ID, "embed"); !errors.Is(err, connection.ErrTypeUnavailable) {
		t.Errorf("err = %v, want ErrTypeUnavailable", err)
	}
}

func TestAFailureOfTheProviderIsAProviderUnavailableWithTheReason(t *testing.T) {
	for name, arrange := range map[string]func(*setup) string{
		"a rejected key": func(s *setup) string {
			s.world.Provider.FailAlways(address, &connection.TryNextError{Reason: connection.FailureReasonRejectedKey})
			return "key rejected"
		},
		"a stop": func(s *setup) string {
			s.world.Provider.FailAlways(address, &connection.StopError{Reason: connection.FailureReasonInvalidRequest})
			return "bad request"
		},
		"an error of no kind": func(s *setup) string {
			s.world.Provider.FailAlways(address, errors.New("rpc error: connection reset"))
			return "unreachable"
		},
		"a plugin that does not start": func(s *setup) string {
			s.world.Source.FailOpen(modelcalltest.ProviderPluginID, errors.New("missing program"))
			return "unreachable"
		},
		"a key that is missing": func(s *setup) string {
			clear(s.world.Keys.Values)
			return "key missing"
		},
	} {
		s := newSetup(t)
		want := arrange(s)

		_, err := s.lists.ListModels(t.Context(), s.entry.ID, "embed")

		if !errors.Is(err, modellist.ErrProviderUnavailable) || !strings.HasSuffix(err.Error(), want) {
			t.Errorf("%s: err = %v, want a provider unavailable for %q", name, err, want)
		}
	}
}

func TestAFailureSetsNoCooldownBecauseAPersonAskedForTheList(t *testing.T) {
	s := newSetup(t)
	s.world.Provider.Fail(address, &connection.TryNextError{Reason: connection.FailureReasonPaymentRequired})
	if _, err := s.lists.ListModels(t.Context(), s.entry.ID, "embed"); err == nil {
		t.Fatal("the ask that the provider failed succeeded")
	}

	if _, err := s.lists.ListModels(t.Context(), s.entry.ID, "embed"); err != nil {
		t.Errorf("the second ask failed: %v", err)
	}
}

func TestACanceledAskReturnsTheContextError(t *testing.T) {
	s := newSetup(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := s.lists.ListReasoningLevels(ctx, s.entry.ID, "m"); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestTheKeyDoesNotLeaveInTheErrorText(t *testing.T) {
	s := newSetup(t)
	s.world.Provider.FailAlways(address, errors.New("HTTP 401 for Bearer sk-one"))
	_, err := s.lists.ListModels(t.Context(), s.entry.ID, "embed")
	if err == nil || strings.Contains(err.Error(), "sk-one") {
		t.Errorf("err = %v", err)
	}
}

func TestAReadLongerThanTheReadTimeoutFailsOnceAsUnavailable(t *testing.T) {
	s := newSetup(t)
	s.world.Provider.Block(address)

	_, err := s.lists.ListModels(t.Context(), s.entry.ID, "complete")

	if !errors.Is(err, modellist.ErrProviderUnavailable) || !strings.Contains(err.Error(), connection.FailureReasonUnavailable.String()) {
		t.Errorf("err = %v, want unavailable with a timeout", err)
	}
	if calls := len(s.world.Provider.Calls()); calls != 1 {
		t.Errorf("the provider was asked %d times, want once", calls)
	}
}
