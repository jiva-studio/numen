package search_test

import (
	"context"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/usecase/search"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// floorSpy records the floor and recipe each search by meaning asks for.
type floorSpy struct {
	mu      sync.Mutex
	floors  []float64
	recipes []string
}

func (s *floorSpy) FindLexical(context.Context, domain.VaultID, string, []domain.SourceKind, int, bool) ([]domain.Passage, error) {
	return nil, nil
}

func (s *floorSpy) GetNamedPassages(context.Context, domain.VaultID, string, []domain.SourceKind, int, bool) ([]domain.Passage, error) {
	return nil, nil
}

func (s *floorSpy) FindNearest(_ context.Context, _ domain.VaultID, recipe string, _ []float32, _ []domain.SourceKind, _ int, floor float64) ([]domain.Passage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.floors = append(s.floors, floor)
	s.recipes = append(s.recipes, recipe)
	return nil, nil
}

func TestSearchResolvesFloorFromTheFirstSourceThatNamesOne(t *testing.T) {
	vault := domain.Vault{ID: "01VAULT"}
	source := func(_ context.Context, v domain.Vault, recipe string) float64 {
		if v.ID != vault.ID || recipe != model.Recipe() {
			t.Errorf("the source was queried for %s under %q", v.ID, recipe)
		}
		return 0.41
	}

	for _, tc := range []struct {
		name      string
		settings  float64
		source    search.FloorSource
		parameter float64
		want      float64
	}{
		{"the query's own floor outranks the rest", 0.6, source, 0.7, 0.7},
		{"the settings file outranks the source", 0.6, source, 0, 0.6},
		{"the source outranks the default", 0, source, 0, 0.41},
		{"the default answers last", 0, nil, 0, search.DefaultFloor},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spy := &floorSpy{}
			searching := search.New(spy, vaultreader.Readers{}, nil, oneWay{direction: newVector(+1)}, tc.settings, nil)
			if tc.source != nil {
				searching = searching.WithFloorSource(tc.source)
			}

			p := search.GetTypingParameters(search.Dense, 5)
			p.Floor = tc.parameter
			if _, err := searching.Execute(t.Context(), vault, "disorder", p); err != nil {
				t.Fatal(err)
			}
			if len(spy.floors) != 1 || spy.floors[0] != tc.want {
				t.Errorf("FindNearest was called with floors %v, want [%v]", spy.floors, tc.want)
			}
			if spy.recipes[0] != model.Recipe() {
				t.Errorf("FindNearest was called under %q, want the model's recipe", spy.recipes[0])
			}
		})
	}
}

func TestTypingParametersLeaveTheFloorToTheSearch(t *testing.T) {
	for _, mode := range []search.Mode{search.Hybrid, search.Lexical, search.Dense, search.ByName} {
		if got := search.GetTypingParameters(mode, 10).Floor; got != 0 {
			t.Errorf("mode %v names the floor %v, want none", mode, got)
		}
	}
}

func TestTheDefaultFloorIsThirtyHundredths(t *testing.T) {
	if search.DefaultFloor != 0.30 {
		t.Errorf("DefaultFloor = %v, want 0.30", search.DefaultFloor)
	}
}

func TestASearchAsksTheFloorSourceOncePerQuery(t *testing.T) {
	vault := domain.Vault{ID: "01VAULT"}
	calls := 0
	searching := search.New(&floorSpy{}, vaultreader.Readers{}, nil, oneWay{direction: newVector(+1)}, 0, nil).
		WithFloorSource(func(context.Context, domain.Vault, string) float64 { calls++; return 0.4 })

	if _, err := searching.Execute(t.Context(), vault, "disorder", search.GetTypingParameters(search.Hybrid, 5)); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Errorf("the source was calls %d times, want once", calls)
	}
}
