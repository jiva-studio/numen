package hostsearch

import (
	"context"
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/embedding"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchpolicy"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	vectorscase "github.com/jiva-studio/numen/libs/core/kernel/usecase/vectors"
)

func getVault() domain.Vault { return domain.Vault{ID: "01VAULT"} }

func getModel() embedding.Model {
	return embedding.Model{Name: "test-model", Dimensions: 4, From: "test"}
}

// stubEmbedder embeds every text as the vector of its length and reports a floor hint.
type stubEmbedder struct {
	mu       sync.Mutex
	model    embedding.Model
	err      error
	short    bool
	hint     float64
	switchTo *embedding.Model
	calls    int
	queried  []string
}

func (e *stubEmbedder) GetModel() embedding.Model {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.model
}

func (e *stubEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.calls++
	if e.err != nil {
		return nil, e.err
	}
	out := make([][]float32, len(texts))
	for i, text := range texts {
		out[i] = []float32{float32(len(text)), 0, 0, 0}
	}
	if e.short {
		out = out[:len(out)-1]
	}
	if e.switchTo != nil {
		e.model = *e.switchTo
	}
	return out, nil
}

// EmbedQueries embeds the texts as queries, and keeps them.
func (e *stubEmbedder) EmbedQueries(ctx context.Context, texts []string) ([][]float32, error) {
	e.mu.Lock()
	e.queried = append(e.queried, texts...)
	e.mu.Unlock()
	return e.Embed(ctx, texts)
}

func (e *stubEmbedder) Concurrency() int { return 1 }
func (e *stubEmbedder) Close() error     { return nil }

func (e *stubEmbedder) GetFloorHint(context.Context) float64 { return e.hint }

// stubSamples answers with n passages of four dimensions, each on a source and a path of its own,
// and records the limit and the recipe it was asked for.
type stubSamples struct {
	vectors      int
	vectorLimit  int
	vectorRecipe string
}

func (s *stubSamples) ListSmallChunkVectors(_ context.Context, _ domain.Vault, recipe string, limit int) ([]searchpolicy.Passage, error) {
	s.vectorLimit, s.vectorRecipe = limit, recipe
	out := make([]searchpolicy.Passage, 0, min(limit, s.vectors))
	for i := range min(limit, s.vectors) {
		out = append(out, searchpolicy.Passage{
			Source: int64(i + 1), Path: fmt.Sprintf("note-%d.md", i+1), Vector: []byte{byte(i), 1, 2, 3},
		})
	}
	return out, nil
}

type stubNearest struct {
	best    float64
	recipes []string
}

func (n *stubNearest) Execute(_ context.Context, _ domain.VaultID, recipe string, _ []float32) (float64, error) {
	n.recipes = append(n.recipes, recipe)
	return n.best, nil
}

type stubProgress struct {
	total, embedded int64
	err             error
}

func (p stubProgress) GetProgress(context.Context, domain.VaultID, string) (int64, int64, error) {
	return p.total, p.embedded, p.err
}

type memoryFloors map[string]float64

func (m memoryFloors) ReadFloor(_ context.Context, v domain.Vault, recipe string) (float64, bool, error) {
	floor, found := m[string(v.ID)+"|"+recipe]
	return floor, found, nil
}

func (m memoryFloors) WriteFloor(_ context.Context, v domain.Vault, recipe string, floor float64) error {
	m[string(v.ID)+"|"+recipe] = floor
	return nil
}

func (m memoryFloors) DeleteFloor(_ context.Context, v domain.Vault, recipe string) error {
	delete(m, string(v.ID)+"|"+recipe)
	return nil
}

type fixture struct {
	search   Search
	embedder *stubEmbedder
	samples  *stubSamples
	nearest  *stubNearest
	floors   memoryFloors
}

func newFixture() *fixture {
	f := &fixture{
		embedder: &stubEmbedder{model: getModel(), hint: 0.4},
		samples:  &stubSamples{vectors: 1500},
		nearest:  &stubNearest{best: 0.7},
		floors:   memoryFloors{},
	}
	f.search = Search{
		Embedder: f.embedder,
		Queries:  vectorscase.NewEmbedQueries(f.embedder, f.embedder),
		Samples:  f.samples,
		Nearest:  f.nearest,
		Progress: stubProgress{total: 10, embedded: 10},
		Policy:   search.NewFloorPolicy(f.floors, func(ctx context.Context) float64 { return f.embedder.GetFloorHint(ctx) }, nil),
	}
	return f
}

func unitQuery(path string) plugin.SimilarityQuery {
	return plugin.SimilarityQuery{Path: path, Vector: []float32{1, 0, 0, 0}}
}

func TestSampleSimilaritiesCapsThePassages(t *testing.T) {
	f := newFixture()
	got, err := f.search.SampleSimilarities(t.Context(), getVault(), getModel().Recipe(), []plugin.SimilarityQuery{unitQuery("")})
	if err != nil {
		t.Fatal(err)
	}
	if f.samples.vectorLimit != plugin.MaxSampledPassages || got.PassageCount != plugin.MaxSampledPassages {
		t.Errorf("asked the index for %d and sampled %d, want %d", f.samples.vectorLimit, got.PassageCount, plugin.MaxSampledPassages)
	}
	if got.SourceCount != plugin.MaxSampledPassages {
		t.Errorf("%d sources, want %d", got.SourceCount, plugin.MaxSampledPassages)
	}
	if len(got.Rows) != 1 || len(got.Rows[0]) != plugin.MaxSampledPassages {
		t.Fatalf("rows %d, want one of %d similarities", len(got.Rows), plugin.MaxSampledPassages)
	}
	// Passage i holds {i, 1, 2, 3}: its cosine with the first axis is i / sqrt(i*i + 14).
	for _, i := range []int{0, 1, 100, 127} {
		want := float64(int8(i)) / math.Sqrt(float64(int8(i))*float64(int8(i))+14)
		if math.Abs(got.Rows[0][i]-want) > 1e-6 {
			t.Errorf("similarity %d is %v, want %v", i, got.Rows[0][i], want)
		}
	}
}

func TestSampleSimilaritiesIsDeterministic(t *testing.T) {
	f := newFixture()
	queries := []plugin.SimilarityQuery{unitQuery("note-3.md"), {Path: "", Vector: []float32{0.5, 0.5, 0.5, 0.5}}}
	first, err := f.search.SampleSimilarities(t.Context(), getVault(), getModel().Recipe(), queries)
	if err != nil {
		t.Fatal(err)
	}
	second, err := f.search.SampleSimilarities(t.Context(), getVault(), getModel().Recipe(), queries)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Error("two samples of one index differ")
	}
}

func TestSampleSimilaritiesExcludesTheNoteOfTheQuery(t *testing.T) {
	f := newFixture()
	f.samples.vectors = 10
	got, err := f.search.SampleSimilarities(t.Context(), getVault(), getModel().Recipe(), []plugin.SimilarityQuery{
		unitQuery("note-4.md"), unitQuery("missing.md"), unitQuery(""),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Rows[0]) != 9 || len(got.Rows[1]) != 10 || len(got.Rows[2]) != 10 {
		t.Errorf("rows of %d, %d and %d similarities, want 9, 10 and 10", len(got.Rows[0]), len(got.Rows[1]), len(got.Rows[2]))
	}
	if got.PassageCount != 10 || got.SourceCount != 10 {
		t.Errorf("%d passages from %d sources, want 10 from 10", got.PassageCount, got.SourceCount)
	}
}

func TestSampleSimilaritiesReturnsAnEmptyRowWithoutAVector(t *testing.T) {
	f := newFixture()
	got, err := f.search.SampleSimilarities(t.Context(), getVault(), getModel().Recipe(), []plugin.SimilarityQuery{{Path: "a.md"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Rows) != 1 || len(got.Rows[0]) != 0 {
		t.Errorf("rows %v, want one empty row", got.Rows)
	}
}

func TestSampleSimilaritiesRejectsAQueryOfAnotherWidth(t *testing.T) {
	f := newFixture()
	_, err := f.search.SampleSimilarities(t.Context(), getVault(), getModel().Recipe(), []plugin.SimilarityQuery{
		{Vector: []float32{1, 0, 0}},
	})
	if !errors.Is(err, plugin.ErrInvalidQuery) {
		t.Errorf("got %v, want ErrInvalidQuery", err)
	}
}

func TestSampleSimilaritiesCapsTheQueries(t *testing.T) {
	f := newFixture()
	queries := make([]plugin.SimilarityQuery, plugin.MaxSampledQueries+1)
	_, err := f.search.SampleSimilarities(t.Context(), getVault(), getModel().Recipe(), queries)
	if !errors.Is(err, plugin.ErrLimitExceeded) {
		t.Errorf("got %v, want ErrLimitExceeded", err)
	}
	if f.samples.vectorRecipe != "" {
		t.Error("a call over the limit reached the index")
	}
}

func TestACallWithARecipeOfAnotherModelIsRefused(t *testing.T) {
	f := newFixture()
	other := "another|recipe"
	ctx := t.Context()

	if _, err := f.search.SampleSimilarities(ctx, getVault(), other, nil); !errors.Is(err, plugin.ErrRecipeMismatch) {
		t.Errorf("sampling under another recipe gave %v", err)
	}
	if _, err := f.search.GetBestSimilarity(ctx, getVault(), other, []float32{1}); !errors.Is(err, plugin.ErrRecipeMismatch) {
		t.Errorf("a best similarity under another recipe gave %v", err)
	}
	if _, err := f.search.GetSearchFloor(ctx, getVault(), other); !errors.Is(err, plugin.ErrRecipeMismatch) {
		t.Errorf("a floor read under another recipe gave %v", err)
	}
	if err := f.search.SetSearchFloor(ctx, getVault(), other, 0.5); !errors.Is(err, plugin.ErrRecipeMismatch) {
		t.Errorf("a floor set under another recipe gave %v", err)
	}
	if len(f.floors) != 0 || len(f.nearest.recipes) != 0 || f.samples.vectorRecipe != "" {
		t.Errorf("a refused call reached the index: floors %v, nearest %v, samples %q", f.floors, f.nearest.recipes, f.samples.vectorRecipe)
	}
}

func TestACallThatNeedsAModelIsRefusedWhereThereIsNone(t *testing.T) {
	f := newFixture()
	f.search.Embedder = nil
	ctx := t.Context()

	if _, err := f.search.GetEmbeddingState(ctx, getVault()); !errors.Is(err, plugin.ErrNoModel) {
		t.Errorf("state gave %v", err)
	}
	if _, err := f.search.EmbedTexts(ctx, []string{"a"}); !errors.Is(err, plugin.ErrNoModel) {
		t.Errorf("embedding gave %v", err)
	}
	if _, err := f.search.SampleSimilarities(ctx, getVault(), "r", nil); !errors.Is(err, plugin.ErrNoModel) {
		t.Errorf("sampling gave %v", err)
	}
	if err := f.search.SetSearchFloor(ctx, getVault(), "r", 0.5); !errors.Is(err, plugin.ErrNoModel) {
		t.Errorf("setting a floor gave %v", err)
	}
}

func TestAFloorOutsideTheRangeIsRefusedAndStoresNothing(t *testing.T) {
	f := newFixture()
	recipe := getModel().Recipe()
	for _, floor := range []float64{-0.01, 1.0001, 7} {
		if err := f.search.SetSearchFloor(t.Context(), getVault(), recipe, floor); !errors.Is(err, plugin.ErrInvalidFloor) {
			t.Errorf("floor %v gave %v, want ErrInvalidFloor", floor, err)
		}
	}
	if len(f.floors) != 0 {
		t.Errorf("a refused floor was stored: %v", f.floors)
	}
	if err := f.search.SetSearchFloor(t.Context(), getVault(), recipe, 1); err != nil {
		t.Errorf("a floor of one gave %v", err)
	}
}

func TestSettingTheStoredFloorAgainChangesNothingAndZeroRemovesIt(t *testing.T) {
	f := newFixture()
	recipe := getModel().Recipe()
	ctx := t.Context()
	for range 3 {
		if err := f.search.SetSearchFloor(ctx, getVault(), recipe, 0.55); err != nil {
			t.Fatal(err)
		}
	}
	got, err := f.search.GetSearchFloor(ctx, getVault(), recipe)
	if err != nil {
		t.Fatal(err)
	}
	if got.Floor != 0.55 || got.Source != plugin.FloorStored || got.Stored != 0.55 || got.Hint != 0.4 || got.Default != search.DefaultFloor {
		t.Errorf("floor %+v", got)
	}
	if len(f.floors) != 1 {
		t.Errorf("%d floors are stored after three identical sets, want one", len(f.floors))
	}

	if err := f.search.SetSearchFloor(ctx, getVault(), recipe, 0); err != nil {
		t.Fatal(err)
	}
	got, err = f.search.GetSearchFloor(ctx, getVault(), recipe)
	if err != nil {
		t.Fatal(err)
	}
	if got.Floor != 0.4 || got.Source != plugin.FloorModelHint || got.Stored != 0 {
		t.Errorf("after removing the stored floor: %+v, want the hint", got)
	}
	f.embedder.hint = 0
	got, err = f.search.GetSearchFloor(ctx, getVault(), recipe)
	if err != nil {
		t.Fatal(err)
	}
	if got.Floor != search.DefaultFloor || got.Source != plugin.FloorDefault {
		t.Errorf("with no hint: %+v, want the default", got)
	}
}

func TestTheEmbeddingStateIsReadFromTheIndexAtEachCall(t *testing.T) {
	f := newFixture()
	ctx := t.Context()

	f.search.Progress = stubProgress{total: 10, embedded: 4}
	state, err := f.search.GetEmbeddingState(ctx, getVault())
	if err != nil {
		t.Fatal(err)
	}
	if state.IsReady() || state.Recipe != getModel().Recipe() || state.Model != "test-model" || state.FloorHint != 0.4 ||
		state.ChunkCount != 10 || state.EmbeddedCount != 4 {
		t.Errorf("state while embedding: %+v", state)
	}

	f.search.Progress = stubProgress{total: 10, embedded: 10}
	for range 2 {
		state, err = f.search.GetEmbeddingState(ctx, getVault())
		if err != nil {
			t.Fatal(err)
		}
		if !state.IsReady() {
			t.Errorf("state once embedded, read again: %+v", state)
		}
	}

	f.search.Progress = stubProgress{}
	if state, err = f.search.GetEmbeddingState(ctx, getVault()); err != nil || state.IsReady() {
		t.Errorf("an empty vault is ready: %+v (%v)", state, err)
	}

	progressErr := errors.New("index closed")
	f.search.Progress = stubProgress{err: progressErr}
	if _, err = f.search.GetEmbeddingState(ctx, getVault()); !errors.Is(err, progressErr) {
		t.Errorf("a progress that cannot be read gave %v", err)
	}
}

func TestTextsAreEmbeddedWithinTheLimits(t *testing.T) {
	f := newFixture()
	ctx := t.Context()

	got, err := f.search.EmbedTexts(ctx, []string{"a", "abc"})
	if err != nil || got.Recipe != getModel().Recipe() || len(got.Vectors) != 2 || got.Vectors[1][0] != 3 {
		t.Fatalf("embedded %+v (%v)", got, err)
	}
	empty, err := f.search.EmbedTexts(ctx, nil)
	if err != nil || len(empty.Vectors) != 0 || empty.Recipe != getModel().Recipe() {
		t.Errorf("no texts gave %+v (%v)", empty, err)
	}

	many := make([]string, plugin.MaxEmbeddedTexts+1)
	if _, err := f.search.EmbedTexts(ctx, many); !errors.Is(err, plugin.ErrLimitExceeded) {
		t.Errorf("%d texts gave %v", len(many), err)
	}
	if _, err := f.search.EmbedTexts(ctx, []string{strings.Repeat("x", plugin.MaxEmbeddedBytes+1)}); !errors.Is(err, plugin.ErrLimitExceeded) {
		t.Errorf("a long text gave %v", err)
	}
	atLimit := make([]string, plugin.MaxEmbeddedTexts)
	for i := range atLimit {
		atLimit[i] = strings.Repeat("x", plugin.MaxEmbeddedBytes)
	}
	if _, err := f.search.EmbedTexts(ctx, atLimit); err != nil {
		t.Errorf("texts at the limit gave %v", err)
	}
}

func TestAnEmbeddingThatFailsOrChangesTheModelIsRefused(t *testing.T) {
	ctx := t.Context()

	f := newFixture()
	f.embedder.err = errors.New("the plugin is not running")
	if _, err := f.search.EmbedTexts(ctx, []string{"a"}); !errors.Is(err, plugin.ErrEmbeddingFailed) {
		t.Errorf("a failed embedding gave %v", err)
	}

	f = newFixture()
	f.embedder.short = true
	if _, err := f.search.EmbedTexts(ctx, []string{"a", "b"}); !errors.Is(err, plugin.ErrEmbeddingFailed) {
		t.Errorf("a short answer gave %v", err)
	}

	f = newFixture()
	changed := embedding.Model{Name: "other-model", Dimensions: 4, From: "test"}
	f.embedder.switchTo = &changed
	if _, err := f.search.EmbedTexts(ctx, []string{"a"}); !errors.Is(err, plugin.ErrRecipeMismatch) {
		t.Errorf("a model that changed during the call gave %v", err)
	}

	f = newFixture()
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	f.embedder.err = context.Canceled
	if _, err := f.search.EmbedTexts(canceled, []string{"a"}); !errors.Is(err, context.Canceled) {
		t.Errorf("a canceled call gave %v, want the cancellation", err)
	}
}

func TestTheBestSimilarityIsReadForTheModelRecipe(t *testing.T) {
	f := newFixture()
	got, err := f.search.GetBestSimilarity(t.Context(), getVault(), getModel().Recipe(), []float32{1, 0, 0, 0})
	if err != nil || got != 0.7 {
		t.Errorf("got %v (%v)", got, err)
	}
	if len(f.nearest.recipes) != 1 || f.nearest.recipes[0] != getModel().Recipe() {
		t.Errorf("the index was asked under %v", f.nearest.recipes)
	}
}

func TestEmbedTextsEmbedsTheTextsAsQueries(t *testing.T) {
	f := newFixture()
	if _, err := f.search.EmbedTexts(t.Context(), []string{"a", "abc"}); err != nil {
		t.Fatal(err)
	}
	if want := []string{"a", "abc"}; !slices.Equal(f.embedder.queried, want) {
		t.Errorf("embedded as queries %q, want %q: the texts of this call are the questions of a plugin", f.embedder.queried, want)
	}
}
