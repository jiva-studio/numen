package rowoptions_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	modelcalldomain "github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/modelcalltest"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/rowoptions"
)

const (
	address            = "https://one.test/v1"
	reasoningField     = "reasoning"
	reasoningOperation = "llm.reasoning"
)

type setup struct {
	world   *modelcalltest.World
	files   *memoryLists
	running *runningProviders
	options *rowoptions.Options
	entry   modelcalltest.Entry
}

func newSetup(t *testing.T) *setup {
	t.Helper()
	w := modelcalltest.New()
	s := &setup{
		world: w, entry: w.AddServer(t, "First", address, "sk-one"),
		files: &memoryLists{byPlugin: map[string][]rowoptions.KeptList{}}, running: &runningProviders{provider: w.Provider},
	}
	w.AddRowField(modelcalltest.ServerTypeID, manifest.ConnectionField{Key: reasoningField, Type: "enum", Title: "Reasoning", OptionsFrom: reasoningOperation})
	s.options = s.restart()
	return s
}

func (s *setup) listModels(ctx context.Context, operations ...string) ([]modelcalldomain.Option, error) {
	return s.options.ListOptions(ctx, s.entry.ID, modelcalltest.ModelOption, map[string]string{modelcalltest.ModelOption: "model-1", reasoningField: "low"}, operations)
}

func (s *setup) listReasoning(ctx context.Context, model string) ([]modelcalldomain.Option, error) {
	row := map[string]string{modelcalltest.ModelOption: model, reasoningField: "low"}
	return s.options.ListOptions(ctx, s.entry.ID, reasoningField, row, []string{modelcalltest.EmbedOperation})
}

func TestListOptionsReturnsTheOptionsTheProviderListsForTheField(t *testing.T) {
	s := newSetup(t)

	options, err := s.listModels(t.Context(), modelcalltest.EmbedOperation)

	want := []modelcalldomain.Option{{Value: "model-1", Title: "Model one", Group: modelcalltest.EmbedOperation}, {Value: "model-2"}}
	if err != nil || !slices.Equal(options, want) {
		t.Errorf("options = %v, %v, want %v", options, err, want)
	}
}

func TestListOptionsCallsTheOperationOfTheFieldWithTheConnectionItsKeyAndTheRowSoFar(t *testing.T) {
	s := newSetup(t)

	if _, err := s.listReasoning(t.Context(), "model-1"); err != nil {
		t.Fatal(err)
	}

	got := s.world.Provider.Calls()[0]
	if got.Operation != reasoningOperation || !maps.Equal(got.Call.Options, map[string]string{modelcalltest.ModelOption: "model-1"}) ||
		got.Call.Secrets["apiKey"] != "sk-one" || got.Call.Fields["baseUrl"] != address ||
		!slices.Equal(got.Operations, []string{modelcalltest.EmbedOperation}) {
		t.Errorf("call = %+v", got)
	}
	if providers := s.world.Source.Opened(); !slices.Equal(providers, []string{modelcalltest.ProviderPluginID}) || s.world.Source.Released() != 1 {
		t.Errorf("opened = %v, released = %d", providers, s.world.Source.Released())
	}
}

func TestAFieldTheTypeListsNoOptionsForHasNone(t *testing.T) {
	s := newSetup(t)

	_, err := s.options.ListOptions(t.Context(), s.entry.ID, "temperature", nil, nil)

	if !errors.Is(err, rowoptions.ErrNoOptions) || len(s.world.Provider.Calls()) != 0 {
		t.Errorf("err = %v, calls = %v, want ErrNoOptions and no call", err, s.world.Provider.Calls())
	}
}

func TestAConnectionThatIsNotInTheListIsNotFound(t *testing.T) {
	s := newSetup(t)
	if _, err := s.options.ListOptions(t.Context(), "c0000dead", modelcalltest.ModelOption, nil, nil); !errors.Is(err, connection.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestAConnectionWhoseTypeIsNotProvidedHasNoOptions(t *testing.T) {
	s := newSetup(t)
	s.world.SetAvailable(modelcalltest.ProviderPluginID, false)
	if _, err := s.listModels(t.Context()); !errors.Is(err, connection.ErrTypeUnavailable) {
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
			return "invalid request"
		},
		"an error of no kind": func(s *setup) string {
			s.world.Provider.FailAlways(address, errors.New("rpc error: connection reset"))
			return "unavailable"
		},
		"a plugin that does not start": func(s *setup) string {
			s.world.Source.FailOpen(modelcalltest.ProviderPluginID, errors.New("missing program"))
			return "unavailable"
		},
		"a key that is missing": func(s *setup) string {
			clear(s.world.Keys.Values)
			return "key missing"
		},
	} {
		s := newSetup(t)
		want := arrange(s)

		_, err := s.listModels(t.Context())

		if !errors.Is(err, rowoptions.ErrProviderUnavailable) || !strings.HasSuffix(err.Error(), want) {
			t.Errorf("%s: err = %v, want a provider unavailable for %q", name, err, want)
		}
	}
}

func TestAFailureSetsNoCooldownBecauseAPersonAskedForTheList(t *testing.T) {
	s := newSetup(t)
	s.world.Provider.Fail(address, &connection.TryNextError{Reason: connection.FailureReasonPaymentRequired})
	if _, err := s.listModels(t.Context()); err == nil {
		t.Fatal("the ask that the provider failed succeeded")
	}

	if _, err := s.listModels(t.Context()); err != nil {
		t.Errorf("the second ask failed: %v", err)
	}
}

func TestACanceledAskReturnsTheContextError(t *testing.T) {
	s := newSetup(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := s.listModels(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestTheKeyDoesNotLeaveInTheErrorText(t *testing.T) {
	s := newSetup(t)
	s.world.Provider.FailAlways(address, errors.New("HTTP 401 for Bearer sk-one"))
	_, err := s.listModels(t.Context())
	if err == nil || strings.Contains(err.Error(), "sk-one") {
		t.Errorf("err = %v", err)
	}
}

func TestAReadLongerThanTheReadTimeoutFailsOnceAsUnavailable(t *testing.T) {
	s := newSetup(t)
	s.world.Provider.Block(address)

	_, err := s.listModels(t.Context())

	if !errors.Is(err, rowoptions.ErrProviderUnavailable) || !strings.Contains(err.Error(), connection.FailureReasonUnavailable.String()) {
		t.Errorf("err = %v, want unavailable", err)
	}
	if calls := len(s.world.Provider.Calls()); calls != 1 {
		t.Errorf("the provider was asked %d times, want once", calls)
	}
}
