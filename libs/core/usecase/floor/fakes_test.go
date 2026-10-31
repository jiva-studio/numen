package floor_test

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/clock"
	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/port/floorport"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

var vault = domain.Vault{ID: "01VAULT"}

// memoryStore keeps floors in a map, by vault and recipe.
type memoryStore struct {
	mu       sync.Mutex
	settings map[string]floorport.Setting
	writes   int
	readErr  error
}

func newMemoryStore() *memoryStore {
	return &memoryStore{settings: map[string]floorport.Setting{}}
}

func key(v domain.Vault, recipe string) string { return string(v.ID) + "|" + recipe }

func (s *memoryStore) ReadFloor(_ context.Context, v domain.Vault, recipe string) (floorport.Setting, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.readErr != nil {
		return floorport.Setting{}, false, s.readErr
	}
	setting, found := s.settings[key(v, recipe)]
	return setting, found, nil
}

func (s *memoryStore) WriteCalibration(_ context.Context, v domain.Vault, recipe string, c floorport.Calibration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	setting := s.settings[key(v, recipe)]
	setting.Calibration = &c
	s.settings[key(v, recipe)] = setting
	s.writes++
	return nil
}

func (s *memoryStore) WriteManualFloor(_ context.Context, v domain.Vault, recipe string, isManual bool, floor float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	setting := s.settings[key(v, recipe)]
	setting.IsManual, setting.ManualFloor = isManual, floor
	s.settings[key(v, recipe)] = setting
	s.writes++
	return nil
}

// stubSamples answers with the titles and passages it was given and counts
// its reads.
type stubSamples struct {
	titles   []floorport.Title
	passages []floorport.Passage
	err      error
}

func (s stubSamples) GetTitles(_ context.Context, _ domain.Vault, limit int) ([]floorport.Title, error) {
	return s.titles[:min(limit, len(s.titles))], s.err
}

func (s stubSamples) GetPassages(_ context.Context, _ domain.Vault, _ string, limit int) ([]floorport.Passage, error) {
	return s.passages[:min(limit, len(s.passages))], s.err
}

// stubEmbedder answers every text with the vector the function gives and
// can change its model between the two reads of a calibration.
type stubEmbedder struct {
	mu      sync.Mutex
	model   embeddingport.EmbeddingModel
	vectors func(text string) []float32
	err     error
	// switchAfterEmbed replaces the model once Embed has answered.
	switchAfterEmbed *embeddingport.EmbeddingModel
	embedded         [][]string
	hint             float64
}

func (e *stubEmbedder) GetModel() embeddingport.EmbeddingModel {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.model
}

func (e *stubEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.embedded = append(e.embedded, texts)
	if e.err != nil {
		return nil, e.err
	}
	out := make([][]float32, len(texts))
	for i, text := range texts {
		out[i] = e.vectors(text)
	}
	if e.switchAfterEmbed != nil {
		e.model = *e.switchAfterEmbed
	}
	return out, nil
}

func (e *stubEmbedder) Concurrency() int { return 1 }
func (e *stubEmbedder) Close() error     { return nil }

func (e *stubEmbedder) GetFloorHint(context.Context) float64 { return e.hint }

var errPluginDown = errors.New("the plugin is not running")

// model is the model every test embeds under.
var model = embeddingport.EmbeddingModel{Name: "test-model", Dimensions: 4, From: "test"}

// stored is the stored form of a vector: a signed byte per dimension, on the
// grid the recipe names.
func stored(values ...int8) []byte {
	out := make([]byte, len(values))
	for i, value := range values {
		out[i] = byte(value)
	}
	return out
}

func newClock(instant time.Time) *clock.Fake { return clock.NewFake(instant) }
