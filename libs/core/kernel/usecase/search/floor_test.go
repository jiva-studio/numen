package search_test

import (
	"context"
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	vectorscase "github.com/jiva-studio/numen/libs/core/kernel/usecase/vectors"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func newFloorFixture() *nearestFake {
	f := &nearestFake{}
	f.holdsSimilarity("weak", 0.35)
	f.holdsSimilarity("faint", 0.2)
	f.holdsSimilarity("fair", 0.5)
	f.holdsSimilarity("great", 0.8)
	return f
}

func chunkIDs(names []string) []domain.ChunkID {
	out := make([]domain.ChunkID, len(names))
	for i, n := range names {
		out[i] = domain.ChunkID(n)
	}
	return out
}

func TestSearchResolvesFloorFromTheFirstSourceThatNamesOne(t *testing.T) {
	vault := domain.Vault{ID: "01VAULT"}
	source := func(_ context.Context, v domain.Vault, recipe string) float64 {
		if v.ID != vault.ID || recipe != getModel().Recipe() {
			t.Errorf("the source was queried for %s under %q", v.ID, recipe)
		}
		return 0.41
	}

	for _, tc := range []struct {
		name      string
		source    search.FloorSource
		parameter float64
		want      []string
	}{
		{"the query's own floor outranks the source", source, 0.7, []string{"great"}},
		{"the source outranks the default", source, 0, []string{"great", "fair"}},
		{"the default answers last", nil, 0, []string{"great", "fair", "weak"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spy := newFloorFixture()
			searching := search.New(
				search.Index{Passages: noWords{}, Nearest: spy},
				vaultreader.Readers{},
				nil,
				oneWay{direction: newVector(+1)},
				vectorscase.NewEmbedQueries(oneWay{direction: newVector(+1)}, nil),
				nil,
			)
			if tc.source != nil {
				searching.Floor = tc.source
			}

			p := search.GetTypingParameters(search.Dense, 5)
			p.Floor = tc.parameter
			if _, err := searching.Execute(t.Context(), vault, "disorder", p); err != nil {
				t.Fatal(err)
			}
			if len(spy.enclosed) != 1 || !slices.Equal(spy.enclosed[0], chunkIDs(tc.want)) {
				t.Errorf("the chunks that stayed above the floor are %v, want %v", spy.enclosed, tc.want)
			}
			if spy.stored[0].Recipe != getModel().Recipe() || spy.coarse[0].Recipe != getModel().Recipe() {
				t.Errorf("the index was asked under %q and %q, want the model's recipe", spy.stored[0].Recipe, spy.coarse[0].Recipe)
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
	searching := search.New(
		search.Index{Passages: noWords{}, Nearest: newFloorFixture()},
		vaultreader.Readers{},
		nil,
		oneWay{direction: newVector(+1)},
		vectorscase.NewEmbedQueries(oneWay{direction: newVector(+1)}, nil),
		nil,
	)
	searching.Floor = func(context.Context, domain.Vault, string) float64 { calls++; return 0.4 }

	if _, err := searching.Execute(t.Context(), vault, "disorder", search.GetTypingParameters(search.Hybrid, 5)); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Errorf("the source was asked %d times, want once", calls)
	}
}

// memoryFloors keeps floors in a map, by vault and recipe.
type memoryFloors struct {
	floors  map[string]float64
	readErr error
	deletes int
}

func newMemoryFloors() *memoryFloors { return &memoryFloors{floors: map[string]float64{}} }

func floorKey(v domain.Vault, recipe string) string { return string(v.ID) + "|" + recipe }

func (m *memoryFloors) ReadFloor(_ context.Context, v domain.Vault, recipe string) (float64, bool, error) {
	if m.readErr != nil {
		return 0, false, m.readErr
	}
	floor, found := m.floors[floorKey(v, recipe)]
	return floor, found, nil
}

func (m *memoryFloors) WriteFloor(_ context.Context, v domain.Vault, recipe string, floor float64) error {
	m.floors[floorKey(v, recipe)] = floor
	return nil
}

func (m *memoryFloors) DeleteFloor(_ context.Context, v domain.Vault, recipe string) error {
	delete(m.floors, floorKey(v, recipe))
	m.deletes++
	return nil
}

func hintOf(value float64) search.FloorHint {
	return func(context.Context) float64 { return value }
}

func TestFloorPolicyPrefersTheStoredFloorToTheHintAndTheHintToTheDefault(t *testing.T) {
	vault := domain.Vault{ID: "01VAULT"}
	recipe := getModel().Recipe()

	for _, tc := range []struct {
		name       string
		stored     float64
		hint       float64
		want       float64
		wantSource search.FloorSourceKind
	}{
		{"the stored floor answers first", 0.55, 0.40, 0.55, search.FloorFromStore},
		{"the hint answers where nothing is stored", 0, 0.40, 0.40, search.FloorFromModel},
		{"the default answers where nothing else does", 0, 0, search.DefaultFloor, search.FloorFromDefault},
		{"a hint above one is no hint", 0, 1.5, search.DefaultFloor, search.FloorFromDefault},
		{"a negative hint is no hint", 0, -0.2, search.DefaultFloor, search.FloorFromDefault},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := newMemoryFloors()
			if tc.stored > 0 {
				store.floors[floorKey(vault, recipe)] = tc.stored
			}
			got := search.NewFloorPolicy(store, hintOf(tc.hint), nil).Get(t.Context(), vault, recipe)
			if got.Floor != tc.want || got.Source != tc.wantSource {
				t.Errorf("got floor %v from %v, want %v from %v", got.Floor, got.Source, tc.want, tc.wantSource)
			}
			if got.Stored != tc.stored || got.Default != search.DefaultFloor {
				t.Errorf("sources stored %v default %v, want %v and %v", got.Stored, got.Default, tc.stored, search.DefaultFloor)
			}
		})
	}
}

func TestFloorPolicyKeepsAFloorPerVaultAndPerRecipe(t *testing.T) {
	store := newMemoryFloors()
	policy := search.NewFloorPolicy(store, nil, nil)
	one := domain.Vault{ID: "01ONE"}
	two := domain.Vault{ID: "01TWO"}
	if err := policy.Set(t.Context(), one, "recipe-a", 0.5); err != nil {
		t.Fatal(err)
	}
	for name, probe := range map[string]struct {
		vault  domain.Vault
		recipe string
	}{
		"another vault":  {two, "recipe-a"},
		"another recipe": {one, "recipe-b"},
	} {
		if got := policy.GetFloor(t.Context(), probe.vault, probe.recipe); got != search.DefaultFloor {
			t.Errorf("%s has floor %v, want the default", name, got)
		}
	}
	if got := policy.GetFloor(t.Context(), one, "recipe-a"); got != 0.5 {
		t.Errorf("the stored floor reads back as %v, want 0.5", got)
	}
}

func TestFloorPolicyIgnoresAStoredFloorOutsideTheRange(t *testing.T) {
	vault := domain.Vault{ID: "01VAULT"}
	store := newMemoryFloors()
	store.floors[floorKey(vault, "r")] = 3
	if got := search.NewFloorPolicy(store, hintOf(0.4), nil).GetFloor(t.Context(), vault, "r"); got != 0.4 {
		t.Errorf("got %v, want the hint", got)
	}
}

func TestFloorPolicyReportsAStoreThatCannotBeReadAndFallsBack(t *testing.T) {
	store := newMemoryFloors()
	store.readErr = errors.New("disk")
	var reported []error
	policy := search.NewFloorPolicy(store, hintOf(0.4), func(err error) { reported = append(reported, err) })

	if got := policy.GetFloor(t.Context(), domain.Vault{ID: "v"}, "r"); got != 0.4 {
		t.Errorf("got %v, want the hint", got)
	}
	if len(reported) != 1 || !errors.Is(reported[0], store.readErr) {
		t.Errorf("reported %v, want the read error once", reported)
	}
}

func TestFloorPolicyReadsNoStoreForAnEmptyRecipe(t *testing.T) {
	store := newMemoryFloors()
	store.readErr = errors.New("must not be read")
	var reported []error
	policy := search.NewFloorPolicy(store, nil, func(err error) { reported = append(reported, err) })
	if got := policy.GetFloor(t.Context(), domain.Vault{ID: "v"}, ""); got != search.DefaultFloor || len(reported) != 0 {
		t.Errorf("got %v with %v reported, want the default and nothing reported", got, reported)
	}
}

func TestFloorPolicySetRejectsAFloorOutsideTheRange(t *testing.T) {
	policy := search.NewFloorPolicy(newMemoryFloors(), nil, nil)
	for _, floor := range []float64{-0.1, 1.01, math.NaN(), math.Inf(1)} {
		if err := policy.Set(t.Context(), domain.Vault{ID: "v"}, "r", floor); !errors.Is(err, search.ErrInvalidFloor) {
			t.Errorf("floor %v gave %v, want ErrInvalidFloor", floor, err)
		}
	}
	if err := policy.Set(t.Context(), domain.Vault{ID: "v"}, "r", 1); err != nil {
		t.Errorf("a floor of one gave %v", err)
	}
}

func TestFloorPolicySetOfZeroRemovesTheStoredFloor(t *testing.T) {
	vault := domain.Vault{ID: "v"}
	store := newMemoryFloors()
	policy := search.NewFloorPolicy(store, nil, nil)
	if err := policy.Set(t.Context(), vault, "r", 0.6); err != nil {
		t.Fatal(err)
	}
	if err := policy.Set(t.Context(), vault, "r", 0); err != nil {
		t.Fatal(err)
	}
	_, found, err := store.ReadFloor(t.Context(), vault, "r")
	if err != nil || found || store.deletes != 1 {
		t.Errorf("the floor is still stored (found %v, %d deletes, %v)", found, store.deletes, err)
	}
}

func TestFloorPolicySetOfTheStoredFloorChangesNothing(t *testing.T) {
	vault := domain.Vault{ID: "v"}
	store := newMemoryFloors()
	policy := search.NewFloorPolicy(store, nil, nil)
	for range 2 {
		if err := policy.Set(t.Context(), vault, "r", 0.6); err != nil {
			t.Fatal(err)
		}
	}
	if got := policy.GetFloor(t.Context(), vault, "r"); got != 0.6 || len(store.floors) != 1 {
		t.Errorf("floor %v with %d stored, want 0.6 and one", got, len(store.floors))
	}
}
