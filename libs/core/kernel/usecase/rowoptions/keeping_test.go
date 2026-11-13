package rowoptions_test

import (
	"context"
	"slices"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	modelcalldomain "github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/modelcalltest"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/rowoptions"
)

func TestAnEmptyOrPartialListIsReturnedAndNotStored(t *testing.T) {
	for name, listed := range map[string]modelcalldomain.ListedOptions{
		"empty":   {},
		"partial": {Options: []modelcalldomain.Option{{Value: "model-1"}}, IsPartial: true},
	} {
		s := newSetup(t)
		s.world.Provider.ScriptOptions(address, listed)

		first, err := s.listModels(t.Context(), modelcalltest.EmbedOperation)
		if err != nil || !slices.Equal(first, listed.Options) {
			t.Fatalf("%s: options = %v, %v, want %v", name, first, err, listed.Options)
		}
		s.mustListModels(t)

		if calls := s.countCalls(); calls != 2 {
			t.Errorf("%s: the provider was asked %d times, want twice", name, calls)
		}
	}
}

func TestAListIsReadAgainOnceItsPluginIsConfiguredOtherwise(t *testing.T) {
	s := newSetup(t)
	s.mustListModels(t)

	s.configs.set(modelcalltest.ProviderPluginID, "executable at another path")
	s.mustListModels(t)

	if calls := s.countCalls(); calls != 2 {
		t.Errorf("the provider was asked %d times, want twice", calls)
	}
}

func TestAListOfAProviderWithAGrantIsReadAgainOnceTheConnectionsItReachesChange(t *testing.T) {
	s := newSetup(t)
	s.grants.set(modelcalltest.ProviderPluginID, "c0000wren@1")
	s.mustListModels(t)
	s.mustListModels(t)

	s.grants.set(modelcalltest.ProviderPluginID, "c0000wren@1,c0000lark@1")
	s.mustListModels(t)

	if calls := s.countCalls(); calls != 2 {
		t.Errorf("the provider was asked %d times, want twice", calls)
	}
}

func TestARefreshAsksThePluginPastItsOwnList(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newSetup(t)
		s.mustListModels(t)
		if _, err := s.options.RefreshOptions(t.Context(), s.entry.ID, modelcalltest.ModelOption, nil, []string{modelcalltest.EmbedOperation}); err != nil {
			t.Fatal(err)
		}
		s.entry = s.world.Edit(t, s.entry, "First renamed")
		s.options.RefreshConnection(s.entry.ID)
		synctest.Wait()

		var fresh []bool
		for _, one := range s.world.Provider.Calls() {
			fresh = append(fresh, one.IsFresh)
		}
		if !slices.Equal(fresh, []bool{false, true, true}) {
			t.Errorf("fresh = %v, want the first listing alone answered from the list of the plugin", fresh)
		}
	})
}

func TestAProviderWithAGrantHasTheLongerReadTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newSetup(t)
		s.grants.set(modelcalltest.ProviderPluginID, "")
		s.world.Provider.Block(address)
		result := make(chan error, 1)
		go func() {
			_, err := s.listModels(t.Context(), modelcalltest.EmbedOperation)
			result <- err
		}()

		time.Sleep(rowoptions.ReadTimeout + time.Second)
		synctest.Wait()
		if len(result) != 0 {
			t.Fatalf("the read ended within the read timeout of a provider without a grant: %v", <-result)
		}
		time.Sleep(rowoptions.GrantedReadTimeout)
		synctest.Wait()
		if len(result) == 0 {
			t.Error("the read did not give up after the read timeout of a provider with a grant")
		}
	})
}

// firstHeldSource opens a provider whose first listing waits for release before it asks the provider of the
// world.
type firstHeldSource struct {
	*modelcalltest.Provider
	release chan struct{}
	listed  atomic.Int32
}

func (s *firstHeldSource) OpenProvider(context.Context, string) (connectionport.Provider, func(), error) {
	return s, func() {}, nil
}

func (s *firstHeldSource) ListOptions(
	ctx context.Context, operation string, call modelcalldomain.ProviderCall, query modelcalldomain.OptionsQuery,
) (modelcalldomain.ListedOptions, error) {
	if s.listed.Add(1) == 1 {
		<-s.release
	}
	return s.Provider.ListOptions(ctx, operation, call, query)
}

func TestAListOfAnEarlierRevisionDoesNotReplaceTheStoredListOfTheNewRevision(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newSetup(t)
		source := &firstHeldSource{Provider: s.world.Provider, release: make(chan struct{})}
		s.options = s.openOver(source)
		done := make(chan error, 1)
		go func() {
			_, err := s.listModels(t.Context(), modelcalltest.EmbedOperation)
			done <- err
		}()
		synctest.Wait()

		s.entry = s.world.Edit(t, s.entry, "First renamed")
		s.mustListModels(t)
		close(source.release)
		if err := <-done; err != nil {
			t.Fatal(err)
		}
		s.mustListModels(t)

		if calls := s.countCalls(); calls != 2 {
			t.Errorf("the provider was asked %d times, want the two reads alone", calls)
		}
	})
}
