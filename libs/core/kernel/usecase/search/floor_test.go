package search_test

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func TestTypingParametersLeaveTheFloorToTheSearch(t *testing.T) {
	for _, mode := range []search.Mode{search.Hybrid, search.Lexical, search.Sources, search.ByName} {
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
