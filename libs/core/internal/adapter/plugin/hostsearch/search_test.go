package hostsearch

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
	"github.com/jiva-studio/numen/libs/core/usecase/search"
	searchpolicyport "github.com/jiva-studio/numen/libs/core/usecase/search/searchpolicycontract"
	embeddingport "github.com/jiva-studio/numen/libs/core/usecase/vectors/embeddingcontract"
)

var vault = domain.Vault{ID: "01VAULT"}

var model = embeddingport.EmbeddingModel{Name: "test-model", Dimensions: 4, From: "test"}

// stubEmbedder embeds every text as the vector of its length and reports a floor hint.
type stubEmbedder struct {
	mu       sync.Mutex
	model    embeddingport.EmbeddingModel
	err      error
	short    bool
	hint     float64
	switchTo *embeddingport.EmbeddingModel
	calls    int
}

func (e *stubEmbedder) GetModel() embeddingport.EmbeddingModel {
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

func (e *stubEmbedder) Concurrency() int { return 1 }
func (e *stubEmbedder) Close() error     { return nil }

func (e *stubEmbedder) GetFloorHint(context.Context) float64 { return e.hint }

// stubSamples answers with n titles and n vectors and records the limit it was asked for.
type stubSamples struct {
	titles       int
	vectors      int
	titleLimit   int
	vectorLimit  int
	vectorRecipe string
}

func (s *stubSamples) ListNoteTitles(_ context.Context, _ domain.Vault, limit int) ([]searchpolicyport.Title, error) {
	s.titleLimit = limit
	out := make([]searchpolicyport.Title, 0, min(limit, s.titles))
	for i := range min(limit, s.titles) {
		out = append(out, searchpolicyport.Title{Source: int64(i + 1), Text: fmt.Sprintf("title %d", i)})
	}
	return out, nil
}

func (s *stubSamples) ListSmallChunkVectors(_ context.Context, _ domain.Vault, recipe string, limit int) ([]searchpolicyport.Passage, error) {
	s.vectorLimit, s.vectorRecipe = limit, recipe
	out := make([]searchpolicyport.Passage, 0, min(limit, s.vectors))
	for i := range min(limit, s.vectors) {
		out = append(out, searchpolicyport.Passage{Source: int64(i + 1), Vector: []byte{byte(i), 1, 2, 3}})
	}
	return out, nil
}

type stubNearest struct {
	best    float64
	recipes []string
}

func (n *stubNearest) GetBestSimilarity(_ context.Context, _ domain.VaultID, recipe string, _ []float32) (float64, error) {
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
		embedder: &stubEmbedder{model: model, hint: 0.4},
		samples:  &stubSamples{titles: 1500, vectors: 1500},
		nearest:  &stubNearest{best: 0.7},
		floors:   memoryFloors{},
	}
	f.search = Search{
		Embedder: f.embedder,
		Samples:  f.samples,
		Nearest:  f.nearest,
		Progress: stubProgress{total: 10, embedded: 10},
		Policy:   search.NewFloorPolicy(f.floors, func(ctx context.Context) float64 { return f.embedder.GetFloorHint(ctx) }, nil),
	}
	return f
}

func TestAListOfTitlesIsCappedAndAZeroLimitIsTheCap(t *testing.T) {
	f := newFixture()
	for _, tc := range []struct {
		limit     int
		wantAsked int
		wantGot   int
	}{
		{limit: 10, wantAsked: 10, wantGot: 10},
		{limit: 0, wantAsked: pluginport.MaxListedTitles, wantGot: pluginport.MaxListedTitles},
		{limit: -3, wantAsked: pluginport.MaxListedTitles, wantGot: pluginport.MaxListedTitles},
		{limit: 100000, wantAsked: pluginport.MaxListedTitles, wantGot: pluginport.MaxListedTitles},
	} {
		got, err := f.search.ListNoteTitles(t.Context(), vault, tc.limit)
		if err != nil {
			t.Fatal(err)
		}
		if f.samples.titleLimit != tc.wantAsked || len(got) != tc.wantGot {
			t.Errorf("limit %d asked the index for %d and returned %d, want %d and %d",
				tc.limit, f.samples.titleLimit, len(got), tc.wantAsked, tc.wantGot)
		}
	}
}

func TestAListOfVectorsIsCappedAndKeepsTheOrderOfTheIndex(t *testing.T) {
	f := newFixture()
	recipe := model.Recipe()

	got, err := f.search.ListSmallChunkVectors(t.Context(), vault, recipe, 100000)
	if err != nil {
		t.Fatal(err)
	}
	if f.samples.vectorLimit != pluginport.MaxListedVectors || len(got) != pluginport.MaxListedVectors {
		t.Errorf("asked the index for %d and returned %d, want %d", f.samples.vectorLimit, len(got), pluginport.MaxListedVectors)
	}
	for i, one := range got {
		if one.Source != int64(i+1) || one.Vector[0] != byte(i) {
			t.Fatalf("entry %d is %+v: the order of the index was not kept", i, one)
		}
	}
	again, err := f.search.ListSmallChunkVectors(t.Context(), vault, recipe, 100000)
	if err != nil {
		t.Fatal(err)
	}
	for i := range got {
		if got[i].Source != again[i].Source {
			t.Fatalf("two reads ordered entry %d differently", i)
		}
	}
}

func TestATitleListKeepsTheOrderOfTheIndex(t *testing.T) {
	f := newFixture()
	got, err := f.search.ListNoteTitles(t.Context(), vault, 5)
	if err != nil {
		t.Fatal(err)
	}
	for i, one := range got {
		if one.Source != int64(i+1) || one.Title != fmt.Sprintf("title %d", i) {
			t.Fatalf("entry %d is %+v", i, one)
		}
	}
}

func TestACallWithARecipeOfAnotherModelIsRefused(t *testing.T) {
	f := newFixture()
	other := "another|recipe"
	ctx := t.Context()

	if _, err := f.search.ListSmallChunkVectors(ctx, vault, other, 10); !errors.Is(err, pluginport.ErrRecipeMismatch) {
		t.Errorf("listing vectors under another recipe gave %v", err)
	}
	if _, err := f.search.GetBestSimilarity(ctx, vault, other, []float32{1}); !errors.Is(err, pluginport.ErrRecipeMismatch) {
		t.Errorf("a best similarity under another recipe gave %v", err)
	}
	if _, err := f.search.GetSearchFloor(ctx, vault, other); !errors.Is(err, pluginport.ErrRecipeMismatch) {
		t.Errorf("a floor read under another recipe gave %v", err)
	}
	if err := f.search.SetSearchFloor(ctx, vault, other, 0.5); !errors.Is(err, pluginport.ErrRecipeMismatch) {
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

	if _, err := f.search.GetEmbeddingState(ctx, vault); !errors.Is(err, pluginport.ErrNoModel) {
		t.Errorf("state gave %v", err)
	}
	if _, err := f.search.EmbedTexts(ctx, []string{"a"}); !errors.Is(err, pluginport.ErrNoModel) {
		t.Errorf("embedding gave %v", err)
	}
	if _, err := f.search.ListSmallChunkVectors(ctx, vault, "r", 1); !errors.Is(err, pluginport.ErrNoModel) {
		t.Errorf("listing vectors gave %v", err)
	}
	if err := f.search.SetSearchFloor(ctx, vault, "r", 0.5); !errors.Is(err, pluginport.ErrNoModel) {
		t.Errorf("setting a floor gave %v", err)
	}
}

func TestAFloorOutsideTheRangeIsRefusedAndStoresNothing(t *testing.T) {
	f := newFixture()
	recipe := model.Recipe()
	for _, floor := range []float64{-0.01, 1.0001, 7} {
		if err := f.search.SetSearchFloor(t.Context(), vault, recipe, floor); !errors.Is(err, pluginport.ErrInvalidFloor) {
			t.Errorf("floor %v gave %v, want ErrInvalidFloor", floor, err)
		}
	}
	if len(f.floors) != 0 {
		t.Errorf("a refused floor was stored: %v", f.floors)
	}
	if err := f.search.SetSearchFloor(t.Context(), vault, recipe, 1); err != nil {
		t.Errorf("a floor of one gave %v", err)
	}
}

func TestSettingTheStoredFloorAgainChangesNothingAndZeroRemovesIt(t *testing.T) {
	f := newFixture()
	recipe := model.Recipe()
	ctx := t.Context()
	for range 3 {
		if err := f.search.SetSearchFloor(ctx, vault, recipe, 0.55); err != nil {
			t.Fatal(err)
		}
	}
	got, err := f.search.GetSearchFloor(ctx, vault, recipe)
	if err != nil {
		t.Fatal(err)
	}
	if got.Floor != 0.55 || got.Source != pluginport.PluginFloorStored || got.Stored != 0.55 || got.Hint != 0.4 || got.Default != search.DefaultFloor {
		t.Errorf("floor %+v", got)
	}
	if len(f.floors) != 1 {
		t.Errorf("%d floors are stored after three identical sets, want one", len(f.floors))
	}

	if err := f.search.SetSearchFloor(ctx, vault, recipe, 0); err != nil {
		t.Fatal(err)
	}
	got, err = f.search.GetSearchFloor(ctx, vault, recipe)
	if err != nil {
		t.Fatal(err)
	}
	if got.Floor != 0.4 || got.Source != pluginport.PluginFloorModelHint || got.Stored != 0 {
		t.Errorf("after removing the stored floor: %+v, want the hint", got)
	}
	f.embedder.hint = 0
	got, err = f.search.GetSearchFloor(ctx, vault, recipe)
	if err != nil {
		t.Fatal(err)
	}
	if got.Floor != search.DefaultFloor || got.Source != pluginport.PluginFloorDefault {
		t.Errorf("with no hint: %+v, want the default", got)
	}
}

func TestTheEmbeddingStateIsReadFromTheIndexAtEachCall(t *testing.T) {
	f := newFixture()
	ctx := t.Context()

	f.search.Progress = stubProgress{total: 10, embedded: 4}
	state, err := f.search.GetEmbeddingState(ctx, vault)
	if err != nil {
		t.Fatal(err)
	}
	if state.IsReady() || state.Recipe != model.Recipe() || state.Model != "test-model" || state.FloorHint != 0.4 ||
		state.ChunkCount != 10 || state.EmbeddedCount != 4 {
		t.Errorf("state while embedding: %+v", state)
	}

	f.search.Progress = stubProgress{total: 10, embedded: 10}
	for range 2 {
		state, err = f.search.GetEmbeddingState(ctx, vault)
		if err != nil {
			t.Fatal(err)
		}
		if !state.IsReady() {
			t.Errorf("state once embedded, read again: %+v", state)
		}
	}

	f.search.Progress = stubProgress{}
	if state, err = f.search.GetEmbeddingState(ctx, vault); err != nil || state.IsReady() {
		t.Errorf("an empty vault is ready: %+v (%v)", state, err)
	}

	progressErr := errors.New("index closed")
	f.search.Progress = stubProgress{err: progressErr}
	if _, err = f.search.GetEmbeddingState(ctx, vault); !errors.Is(err, progressErr) {
		t.Errorf("a progress that cannot be read gave %v", err)
	}
}

func TestTextsAreEmbeddedWithinTheLimits(t *testing.T) {
	f := newFixture()
	ctx := t.Context()

	got, err := f.search.EmbedTexts(ctx, []string{"a", "abc"})
	if err != nil || got.Recipe != model.Recipe() || len(got.Vectors) != 2 || got.Vectors[1][0] != 3 {
		t.Fatalf("embedded %+v (%v)", got, err)
	}
	empty, err := f.search.EmbedTexts(ctx, nil)
	if err != nil || len(empty.Vectors) != 0 || empty.Recipe != model.Recipe() {
		t.Errorf("no texts gave %+v (%v)", empty, err)
	}

	many := make([]string, pluginport.MaxEmbeddedTexts+1)
	if _, err := f.search.EmbedTexts(ctx, many); !errors.Is(err, pluginport.ErrLimitExceeded) {
		t.Errorf("%d texts gave %v", len(many), err)
	}
	if _, err := f.search.EmbedTexts(ctx, []string{strings.Repeat("x", pluginport.MaxEmbeddedBytes+1)}); !errors.Is(err, pluginport.ErrLimitExceeded) {
		t.Errorf("a long text gave %v", err)
	}
	atLimit := make([]string, pluginport.MaxEmbeddedTexts)
	for i := range atLimit {
		atLimit[i] = strings.Repeat("x", pluginport.MaxEmbeddedBytes)
	}
	if _, err := f.search.EmbedTexts(ctx, atLimit); err != nil {
		t.Errorf("texts at the limit gave %v", err)
	}
}

func TestAnEmbeddingThatFailsOrChangesTheModelIsRefused(t *testing.T) {
	ctx := t.Context()

	f := newFixture()
	f.embedder.err = errors.New("the plugin is not running")
	if _, err := f.search.EmbedTexts(ctx, []string{"a"}); !errors.Is(err, pluginport.ErrEmbeddingFailed) {
		t.Errorf("a failed embedding gave %v", err)
	}

	f = newFixture()
	f.embedder.short = true
	if _, err := f.search.EmbedTexts(ctx, []string{"a", "b"}); !errors.Is(err, pluginport.ErrEmbeddingFailed) {
		t.Errorf("a short answer gave %v", err)
	}

	f = newFixture()
	changed := embeddingport.EmbeddingModel{Name: "other-model", Dimensions: 4, From: "test"}
	f.embedder.switchTo = &changed
	if _, err := f.search.EmbedTexts(ctx, []string{"a"}); !errors.Is(err, pluginport.ErrRecipeMismatch) {
		t.Errorf("a model that changed during the call gave %v", err)
	}

	f = newFixture()
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	f.embedder.err = context.Canceled
	if _, err := f.search.EmbedTexts(cancelled, []string{"a"}); !errors.Is(err, context.Canceled) {
		t.Errorf("a cancelled call gave %v, want the cancellation", err)
	}
}

func TestTheBestSimilarityIsReadForTheModelRecipe(t *testing.T) {
	f := newFixture()
	got, err := f.search.GetBestSimilarity(t.Context(), vault, model.Recipe(), []float32{1, 0, 0, 0})
	if err != nil || got != 0.7 {
		t.Errorf("got %v (%v)", got, err)
	}
	if len(f.nearest.recipes) != 1 || f.nearest.recipes[0] != model.Recipe() {
		t.Errorf("the index was asked under %v", f.nearest.recipes)
	}
}
