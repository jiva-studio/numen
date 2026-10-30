package container

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/usecase/floor"
	"github.com/jiva-studio/numen/libs/core/usecase/search"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

var floorModel = embeddingport.EmbeddingModel{Name: "floor-model", Dimensions: 4, From: "test"}

// noHintEmbedder reports no floor hint.
type noHintEmbedder struct{}

func (noHintEmbedder) GetModel() embeddingport.EmbeddingModel { return floorModel }
func (noHintEmbedder) Concurrency() int                       { return 1 }
func (noHintEmbedder) Close() error                           { return nil }
func (noHintEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range out {
		out[i] = []float32{1, 0, 0, 0}
	}
	return out, nil
}

// floorHintEmbedder reports a floor hint for its model.
type floorHintEmbedder struct {
	noHintEmbedder
	hint float64
}

func (e floorHintEmbedder) GetFloorHint(context.Context) float64 { return e.hint }

// nearestSpy records the floor of each search by meaning.
type nearestSpy struct {
	mu     sync.Mutex
	floors []float64
}

func (s *nearestSpy) FindLexical(context.Context, domain.VaultID, string, []domain.SourceKind, int, bool) ([]domain.Passage, error) {
	return nil, nil
}

func (s *nearestSpy) GetNamedPassages(context.Context, domain.VaultID, string, []domain.SourceKind, int, bool) ([]domain.Passage, error) {
	return nil, nil
}

func (s *nearestSpy) FindNearest(_ context.Context, _ domain.VaultID, _ string, _ []float32, _ []domain.SourceKind, _ int, floor float64) ([]domain.Passage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.floors = append(s.floors, floor)
	return nil, nil
}

func TestSearchResolvesFloorFromSettingsModelDefault(t *testing.T) {
	vault := domain.Vault{ID: "01VAULT"}
	for _, tc := range []struct {
		name     string
		file     float64
		embedder embeddingport.Embedder
		want     float64
	}{
		{"the settings file", 0.62, floorHintEmbedder{hint: 0.4}, 0.62},
		{"the model's hint", 0, floorHintEmbedder{hint: 0.4}, 0.4},
		{"no hint", 0, noHintEmbedder{}, search.DefaultFloor},
		{"a hint of zero", 0, floorHintEmbedder{}, search.DefaultFloor},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spy := &nearestSpy{}
			cfg := Config{Embedding: EmbeddingConfig{Floor: tc.file}}
			searching := cfg.NewSearchOver(spy, tc.embedder, nil)

			if _, err := searching.Execute(t.Context(), vault, "entropy", search.GetTypingParameters(search.Dense, 5)); err != nil {
				t.Fatal(err)
			}
			if len(spy.floors) != 1 || spy.floors[0] != tc.want {
				t.Errorf("FindNearest was asked with floors %v, want [%v]", spy.floors, tc.want)
			}
		})
	}
}

func TestFloorsOverTheIndexFollowTheOrderOfSources(t *testing.T) {
	ctx := t.Context()
	vault := domain.Vault{ID: "01VAULT"}
	cfg := Config{IndexPath: filepath.Join(t.TempDir(), "index.db")}
	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)

	embedder := floorHintEmbedder{hint: 0.4}
	floors := cfg.NewFloors(db, embedder, nil)
	recipe := floorModel.Recipe()

	if got := floors.Get.Execute(ctx, vault); got.Source != floor.SourceModel || got.Floor != 0.4 || got.Model != "floor-model" {
		t.Fatalf("before anything is kept: %+v", got)
	}

	if err := floors.Set.Execute(ctx, vault, true, 0.7); err != nil {
		t.Fatal(err)
	}
	if got := floors.Resolve.GetFloor(ctx, vault, recipe); got != 0.7 {
		t.Errorf("with a manual floor: %v, want 0.7", got)
	}
	if err := floors.Set.Execute(ctx, vault, false, 0); err != nil {
		t.Fatal(err)
	}
	if got := floors.Resolve.GetFloor(ctx, vault, recipe); got != 0.4 {
		t.Errorf("with the manual floor cleared: %v, want the model's 0.4", got)
	}

	fileFloors := Config{IndexPath: cfg.IndexPath, Embedding: EmbeddingConfig{Floor: 0.9}}.NewFloors(db, embedder, nil)
	if got := fileFloors.Get.Execute(ctx, vault); got.Source != floor.SourceFile || got.Floor != 0.9 {
		t.Errorf("with a settings floor: %+v", got)
	}
}

func TestAFloorHintIsReadOnlyFromAnEmbedderThatReportsOne(t *testing.T) {
	if getFloorHint(nil) != nil || getFloorHint(noHintEmbedder{}) != nil {
		t.Error("an embedder with no hint got a hint function")
	}
	hint := getFloorHint(floorHintEmbedder{hint: 0.33})
	if hint == nil || hint(t.Context()) != 0.33 {
		t.Errorf("hint = %v", hint)
	}
}
