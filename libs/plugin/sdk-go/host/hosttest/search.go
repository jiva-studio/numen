package hosttest

import (
	"context"
	"fmt"
	"math"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// defaultSearchFloor is the floor a search applies where no other source answers.
const defaultSearchFloor = 0.30

// SearchModel is the embedding model the host uses, as the fake reports it.
//
//numen:api
type SearchModel struct {
	Name   string
	Recipe string
	// FloorHint is the starting floor the plugin of the model reports, zero when it has none.
	FloorHint float64
}

// Search is the search index and the search floor of the vaults in memory, answering as the host
// does: every call that names a recipe checks it against the model in use, a sample is capped, and
// a floor outside the range is refused. A call on a vault the host does not know returns
// ErrUnknownVault. With returns the search of another vault of the same host.
type Search struct {
	all   *searchHost
	vault domain.VaultID
}

// searchHost is the index of every vault one host keeps.
type searchHost struct {
	mu      sync.Mutex
	current domain.VaultID
	hasShow bool
	vaults  map[domain.VaultID]*searchVault
	model   *SearchModel
	embed   func(text string) []float32
	failure error
	denied  map[string]bool
	embeds  [][]string
	cosine  func(query, chunk []float32) float64
}

// SampledChunk is a small chunk of the index in the fake: the vector it carries and the path of the
// note it belongs to.
type SampledChunk struct {
	Path   string
	Vector []float32
}

// searchVault is what the index holds of one vault.
type searchVault struct {
	chunks        map[string][]SampledChunk
	chunkCount    int64
	embeddedCount int64
	stats         host.IndexStats
	indexed       []host.SearchResult
	floors        map[string]float64
	setCount      int
}

var (
	_ host.SearchIndex  = (*Search)(nil)
	_ host.SearchPolicy = (*Search)(nil)
)

// NewSearch is the search of a host that shows the vault given, and also knows the others. The host has
// no model until SetModel is called.
func NewSearch(current domain.VaultID, others ...domain.VaultID) *Search {
	all := &searchHost{current: current, hasShow: true, vaults: map[domain.VaultID]*searchVault{}, denied: map[string]bool{}}
	for _, id := range append([]domain.VaultID{current}, others...) {
		all.vaults[id] = &searchVault{chunks: map[string][]SampledChunk{}, floors: map[string]float64{}}
	}
	return &Search{all: all}
}

// With is the search of the vault of the identity. The vault is unknown to the host unless NewSearch named it.
func (s *Search) With(id domain.VaultID) *Search {
	return &Search{all: s.all, vault: id}
}

// SetModel makes the model the one in use.
func (s *Search) SetModel(model SearchModel) {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	s.all.model = &model
}

// RemoveModel leaves the host without a model.
func (s *Search) RemoveModel() {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	s.all.model = nil
}

// SetEmbedder is how the model turns a text into a vector.
func (s *Search) SetEmbedder(embed func(text string) []float32) {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	s.all.embed = embed
}

// FailEmbedding makes the model fail every embedding with the error, until it is called with nil.
func (s *Search) FailEmbedding(err error) {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	s.all.failure = err
}

// HideVault makes the host show no vault, so a call that names none returns ErrNoVault.
func (s *Search) HideVault() {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	s.all.hasShow = false
}

// Deny makes every call that needs one of the permissions return ErrPermissionDenied.
func (s *Search) Deny(permissions ...string) {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	for _, permission := range permissions {
		s.all.denied[permission] = true
	}
}

// SetSampledChunks is the small chunks of the vault under the recipe, in the order the index lists them.
func (s *Search) SetSampledChunks(recipe string, chunks []SampledChunk) {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	if one, ok := s.lookupVaultLocked(); ok {
		one.chunks[recipe] = slices.Clone(chunks)
	}
}

// SetCosine is how the fake compares a query with a chunk. The default is the cosine of the angle
// between the two vectors.
func (s *Search) SetCosine(cosine func(query, chunk []float32) float64) {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	s.all.cosine = cosine
}

// SetProgress is how many chunks of the vault can carry a vector, and how many do.
func (s *Search) SetProgress(chunkCount, embeddedCount int64) {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	if one, ok := s.lookupVaultLocked(); ok {
		one.chunkCount, one.embeddedCount = chunkCount, embeddedCount
	}
}

// SetIndexStats is the sources, extensions and sizes of the vault that GetIndexStats reports. The
// model and the counts of chunks and vectors are those of SetModel and SetProgress.
func (s *Search) SetIndexStats(stats host.IndexStats) {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	if one, ok := s.lookupVaultLocked(); ok {
		one.stats = stats
	}
}

// StoredFloor is the floor stored for the vault under the recipe, and false where none is.
func (s *Search) StoredFloor(recipe string) (float64, bool) {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	one, ok := s.lookupVaultLocked()
	if !ok {
		return 0, false
	}
	floor, found := one.floors[recipe]
	return floor, found
}

// FloorWrites is how many times a floor was stored or removed for the vault.
func (s *Search) FloorWrites() int {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	if one, ok := s.lookupVaultLocked(); ok {
		return one.setCount
	}
	return 0
}

// EmbeddedBatches is the texts of each embedding call the model answered, in order.
func (s *Search) EmbeddedBatches() [][]string {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	return slices.Clone(s.all.embeds)
}

func (s *Search) lookupVaultLocked() (*searchVault, bool) {
	id := s.vault
	if id == "" {
		if !s.all.hasShow {
			return nil, false
		}
		id = s.all.current
	}
	one, ok := s.all.vaults[id]
	return one, ok
}

func (s *Search) openVaultLocked(permission string) (*searchVault, error) {
	if s.all.denied[permission] {
		return nil, fmt.Errorf("%w: %s", host.ErrPermissionDenied, permission)
	}
	if s.vault == "" && !s.all.hasShow {
		return nil, host.ErrNoVault
	}
	one, ok := s.lookupVaultLocked()
	if !ok {
		return nil, host.ErrUnknownVault
	}
	return one, nil
}

func (s *Search) checkRecipeLocked(recipe string) error {
	if s.all.model == nil {
		return host.ErrNoModel
	}
	if s.all.model.Recipe != recipe {
		return host.ErrRecipeMismatch
	}
	return nil
}

func (s *Search) GetEmbeddingState(_ context.Context) (host.EmbeddingState, error) {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	one, err := s.openVaultLocked(host.PermissionIndexRead)
	if err != nil {
		return host.EmbeddingState{}, err
	}
	if s.all.model == nil {
		return host.EmbeddingState{}, host.ErrNoModel
	}
	return host.EmbeddingState{
		Recipe:        s.all.model.Recipe,
		Model:         s.all.model.Name,
		FloorHint:     s.all.model.FloorHint,
		ChunkCount:    one.chunkCount,
		EmbeddedCount: one.embeddedCount,
		IsReady:       one.chunkCount > 0 && one.embeddedCount >= one.chunkCount,
	}, nil
}

func (s *Search) GetIndexStats(_ context.Context) (host.IndexStats, error) {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	one, err := s.openVaultLocked(host.PermissionIndexRead)
	if err != nil {
		return host.IndexStats{}, err
	}
	if s.all.model == nil {
		return host.IndexStats{}, host.ErrNoModel
	}
	stats := one.stats
	stats.Extensions = slices.Clone(stats.Extensions)
	stats.Recipe, stats.Model = s.all.model.Recipe, s.all.model.Name
	stats.ChunkCount, stats.VectorCount = one.chunkCount, one.embeddedCount
	return stats, nil
}

func (s *Search) EmbedTexts(_ context.Context, texts []string) (host.EmbeddedTexts, error) {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	if s.all.denied[host.PermissionIndexRead] {
		return host.EmbeddedTexts{}, fmt.Errorf("%w: %s", host.ErrPermissionDenied, host.PermissionIndexRead)
	}
	if s.all.model == nil {
		return host.EmbeddedTexts{}, host.ErrNoModel
	}
	if len(texts) > host.MaxEmbeddedTexts {
		return host.EmbeddedTexts{}, host.ErrLimitExceeded
	}
	for _, text := range texts {
		if len(text) > host.MaxEmbeddedBytes {
			return host.EmbeddedTexts{}, host.ErrLimitExceeded
		}
	}
	if s.all.failure != nil {
		return host.EmbeddedTexts{}, fmt.Errorf("%w: %w", host.ErrEmbeddingFailed, s.all.failure)
	}
	s.all.embeds = append(s.all.embeds, slices.Clone(texts))
	vectors := make([][]float32, len(texts))
	for i, text := range texts {
		if s.all.embed == nil {
			return host.EmbeddedTexts{}, fmt.Errorf("%w: the fake has no embedder", host.ErrEmbeddingFailed)
		}
		vectors[i] = s.all.embed(text)
	}
	return host.EmbeddedTexts{Recipe: s.all.model.Recipe, Vectors: vectors}, nil
}

func (s *Search) SampleSimilarities(_ context.Context, recipe string, queries []host.SimilarityQuery) (host.SimilaritySample, error) {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	one, err := s.openVaultLocked(host.PermissionIndexRead)
	if err != nil {
		return host.SimilaritySample{}, err
	}
	if err := s.checkRecipeLocked(recipe); err != nil {
		return host.SimilaritySample{}, err
	}
	if len(queries) > host.MaxSampledQueries {
		return host.SimilaritySample{}, host.ErrLimitExceeded
	}
	chunks := one.chunks[recipe]
	chunks = chunks[:min(host.MaxSampledChunks, len(chunks))]
	paths := map[string]struct{}{}
	for _, chunk := range chunks {
		paths[chunk.Path] = struct{}{}
	}
	rows := make([][]float64, len(queries))
	for i, query := range queries {
		row, err := s.compareRowLocked(query, chunks)
		if err != nil {
			return host.SimilaritySample{}, err
		}
		rows[i] = row
	}
	return host.SimilaritySample{ChunkCount: len(chunks), SourceCount: len(paths), Rows: rows}, nil
}

func (s *Search) GetBestSimilarity(_ context.Context, recipe string, query []float32) (float64, error) {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	one, err := s.openVaultLocked(host.PermissionIndexRead)
	if err != nil {
		return 0, err
	}
	if err := s.checkRecipeLocked(recipe); err != nil {
		return 0, err
	}
	best, found := 0.0, false
	for _, chunk := range one.chunks[recipe] {
		similarity := s.compareLocked(query, chunk.Vector)
		if !found || similarity > best {
			best, found = similarity, true
		}
	}
	return best, nil
}

func (s *Search) compareRowLocked(query host.SimilarityQuery, chunks []SampledChunk) ([]float64, error) {
	if len(query.Vector) == 0 {
		return nil, nil
	}
	var row []float64
	for _, chunk := range chunks {
		if query.Path != "" && chunk.Path == query.Path {
			continue
		}
		if len(chunk.Vector) != len(query.Vector) {
			return nil, host.ErrInvalidQuery
		}
		row = append(row, s.compareLocked(query.Vector, chunk.Vector))
	}
	return row, nil
}

func (s *Search) compareLocked(query, chunk []float32) float64 {
	if s.all.cosine != nil {
		return s.all.cosine(query, chunk)
	}
	return cosineOf(query, chunk)
}

func (s *Search) GetSearchFloor(_ context.Context, recipe string) (host.SearchFloor, error) {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	one, err := s.openVaultLocked(host.PermissionSearchPolicy)
	if err != nil {
		return host.SearchFloor{}, err
	}
	if err := s.checkRecipeLocked(recipe); err != nil {
		return host.SearchFloor{}, err
	}
	out := host.SearchFloor{Default: defaultSearchFloor, Stored: one.floors[recipe]}
	if hint := s.all.model.FloorHint; hint > 0 && hint <= 1 {
		out.Hint = hint
	}
	switch {
	case out.Stored > 0:
		out.Floor, out.Source = out.Stored, host.FloorFromStore
	case out.Hint > 0:
		out.Floor, out.Source = out.Hint, host.FloorFromModelHint
	default:
		out.Floor, out.Source = defaultSearchFloor, host.FloorFromDefault
	}
	return out, nil
}

func (s *Search) SetSearchFloor(_ context.Context, recipe string, floor float64) error {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	one, err := s.openVaultLocked(host.PermissionSearchPolicy)
	if err != nil {
		return err
	}
	if err := s.checkRecipeLocked(recipe); err != nil {
		return err
	}
	if floor != 0 && !(floor > 0 && floor <= 1) {
		return host.ErrInvalidFloor
	}
	if floor == 0 {
		delete(one.floors, recipe)
	} else {
		one.floors[recipe] = floor
	}
	one.setCount++
	return nil
}

func cosineOf(query, chunk []float32) float64 {
	if len(query) != len(chunk) || len(query) == 0 {
		return 0
	}
	var dot, left, right float64
	for i, q := range query {
		p := float64(chunk[i])
		dot += float64(q) * p
		left += float64(q) * float64(q)
		right += p * p
	}
	if left == 0 || right == 0 {
		return 0
	}
	return dot / math.Sqrt(left*right)
}
