package hosttest

import (
	"context"
	"fmt"
	"math"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
)

// defaultSearchFloor is the floor a search applies where no other source answers.
const defaultSearchFloor = 0.30

// SearchModel is the embedding model the host uses, as the fake reports it.
type SearchModel struct {
	Name   string
	Recipe string
	// FloorHint is the starting floor the plugin of the model reports, zero when it has none.
	FloorHint float64
}

// Search is the search index and the search floor of the vaults in memory, answering as the host
// does: every call that names a recipe checks it against the model in use, the lists are capped, and
// a floor outside the range is refused. A call on a vault the host does not know returns
// ErrUnknownVault. With returns the search of another vault of the same host.
type Search struct {
	all   *searchHost
	vault host.VaultID
}

// searchHost is the index of every vault one host keeps.
type searchHost struct {
	mu      sync.Mutex
	current host.VaultID
	hasShow bool
	vaults  map[host.VaultID]*searchVault
	model   *SearchModel
	embed   func(text string) []float32
	failure error
	denied  map[string]bool
	embeds  [][]string
}

// searchVault is what the index holds of one vault.
type searchVault struct {
	titles        []host.NoteTitle
	vectors       map[string][]host.ChunkVector
	chunkCount    int64
	embeddedCount int64
	floors        map[string]float64
	setCount      int
}

var _ host.Search = (*Search)(nil)

// NewSearch is the search of a host that shows the vault given, and also knows the others. The host has
// no model until SetModel is called.
func NewSearch(current host.VaultID, others ...host.VaultID) *Search {
	all := &searchHost{current: current, hasShow: true, vaults: map[host.VaultID]*searchVault{}, denied: map[string]bool{}}
	for _, id := range append([]host.VaultID{current}, others...) {
		all.vaults[id] = &searchVault{vectors: map[string][]host.ChunkVector{}, floors: map[string]float64{}}
	}
	return &Search{all: all}
}

// With is the search of the vault of the identity. The vault is unknown to the host unless NewSearch named it.
func (s *Search) With(id host.VaultID) host.Search {
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

// SetNoteTitles is the titles the index holds of the vault, in the order the index lists them.
func (s *Search) SetNoteTitles(titles []host.NoteTitle) {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	if one, ok := s.lookupLocked(); ok {
		one.titles = slices.Clone(titles)
	}
}

// SetChunkVectors is the stored vectors of the small chunks of the vault under the recipe, in the
// order the index lists them.
func (s *Search) SetChunkVectors(recipe string, vectors []host.ChunkVector) {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	if one, ok := s.lookupLocked(); ok {
		one.vectors[recipe] = slices.Clone(vectors)
	}
}

// SetProgress is how many chunks of the vault can carry a vector, and how many do.
func (s *Search) SetProgress(chunkCount, embeddedCount int64) {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	if one, ok := s.lookupLocked(); ok {
		one.chunkCount, one.embeddedCount = chunkCount, embeddedCount
	}
}

// StoredFloor is the floor stored for the vault under the recipe, and false where none is.
func (s *Search) StoredFloor(recipe string) (float64, bool) {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	one, ok := s.lookupLocked()
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
	if one, ok := s.lookupLocked(); ok {
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

// lookupLocked is the vault the search works, and false where the host does not know it or shows none.
func (s *Search) lookupLocked() (*searchVault, bool) {
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

// openLocked authorizes a call and finds the vault it works.
func (s *Search) openLocked(permission string) (*searchVault, error) {
	if s.all.denied[permission] {
		return nil, fmt.Errorf("%w: %s", host.ErrPermissionDenied, permission)
	}
	if s.vault == "" && !s.all.hasShow {
		return nil, host.ErrNoVault
	}
	one, ok := s.lookupLocked()
	if !ok {
		return nil, host.ErrUnknownVault
	}
	return one, nil
}

// checkRecipeLocked returns nil when the recipe is the one of the model in use.
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
	one, err := s.openLocked(host.PermissionIndexRead)
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

func (s *Search) ListNoteTitles(_ context.Context, limit int) ([]host.NoteTitle, error) {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	one, err := s.openLocked(host.PermissionIndexRead)
	if err != nil {
		return nil, err
	}
	return slices.Clone(one.titles[:min(clampSearchLimit(limit, host.MaxListedTitles), len(one.titles))]), nil
}

func (s *Search) ListSmallChunkVectors(_ context.Context, recipe string, limit int) ([]host.ChunkVector, error) {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	one, err := s.openLocked(host.PermissionIndexRead)
	if err != nil {
		return nil, err
	}
	if err := s.checkRecipeLocked(recipe); err != nil {
		return nil, err
	}
	chunks := one.vectors[recipe]
	return slices.Clone(chunks[:min(clampSearchLimit(limit, host.MaxListedVectors), len(chunks))]), nil
}

func (s *Search) GetBestSimilarity(_ context.Context, recipe string, query []float32) (float64, error) {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	one, err := s.openLocked(host.PermissionIndexRead)
	if err != nil {
		return 0, err
	}
	if err := s.checkRecipeLocked(recipe); err != nil {
		return 0, err
	}
	best, found := 0.0, false
	for _, chunk := range one.vectors[recipe] {
		similarity := cosineOfStored(query, chunk.Vector)
		if !found || similarity > best {
			best, found = similarity, true
		}
	}
	return best, nil
}

func (s *Search) GetSearchFloor(_ context.Context, recipe string) (host.SearchFloor, error) {
	s.all.mu.Lock()
	defer s.all.mu.Unlock()
	one, err := s.openLocked(host.PermissionSearchPolicy)
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
	one, err := s.openLocked(host.PermissionSearchPolicy)
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

// clampSearchLimit is the limit a list is read with: the maximum for a limit of zero or less, or above it.
func clampSearchLimit(limit, maximum int) int {
	if limit <= 0 || limit > maximum {
		return maximum
	}
	return limit
}

// cosineOfStored is the cosine of the angle between a query vector and a stored vector of signed bytes.
func cosineOfStored(query []float32, stored []byte) float64 {
	if len(query) != len(stored) || len(query) == 0 {
		return 0
	}
	var dot, left, right float64
	for i, q := range query {
		s := float64(int8(stored[i]))
		dot += float64(q) * s
		left += float64(q) * float64(q)
		right += s * s
	}
	if left == 0 || right == 0 {
		return 0
	}
	return dot / math.Sqrt(left*right)
}
