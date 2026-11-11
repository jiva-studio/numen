package relevance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// defaultFloor is the floor shown where the installation has no model to take one from. The host
// applies the same value where no other source answers.
const defaultFloor = 0.30

// Source is where the floor in force comes from.
type Source int

const (
	// SourceDefault: no other source answered.
	SourceDefault Source = iota + 1
	// SourceManual: the person set the floor for the model in use.
	SourceManual
	// SourceCalibrated: measured on the vault with the model in use.
	SourceCalibrated
	// SourceModel: the starting floor the model's plugin reports.
	SourceModel
)

// State is the floor in force for one vault under the model in use, and what it was derived from.
type State struct {
	Floor          float64
	Source         Source
	Model          string
	AutomaticFloor float64
	IsManual       bool
	ManualFloor    float64
	Calibration    *Calibration
}

// ProbeResult is what a probe measured.
type ProbeResult struct {
	BestSimilarity float64
	Floor          float64
	DoesPass       bool
}

// Host is the part of the host the floors work through: the search index, the notes of the vault and
// the records the plugin keeps in its own folder.
type Host struct {
	Search func(domain.VaultID) Search
	Vaults func(domain.VaultID) host.NoteLookup
	Store  func(domain.VaultID) Records
}

// Search is the calls the floors make on the search of one vault, each through the narrow interface it needs.
type Search struct {
	Index  host.SearchIndex
	Policy host.SearchPolicy
}

// Floors measures, sets and applies the search floor of the vaults.
type Floors struct {
	host       Host
	calibrator Calibrator

	mu sync.Mutex
}

// New returns the floors over the host, with measurements stamped by the clock.
func New(h Host, clock Clock) *Floors {
	return &Floors{host: h, calibrator: NewCalibrator(clock), mu: sync.Mutex{}}
}

// scope is the search, the notes and the records of one vault.
type scope struct {
	search  Search
	notes   host.NoteLookup
	records Records
}

func (f *Floors) open(vault domain.VaultID) scope {
	return scope{search: f.host.Search(vault), notes: f.host.Vaults(vault), records: f.host.Store(vault)}
}

// Get returns the floor in force for the vault. An installation with no model has the default floor.
func (f *Floors) Get(ctx context.Context, vault domain.VaultID) (State, error) {
	s := f.open(vault)
	state, err := s.search.Index.GetEmbeddingState(ctx)
	if errors.Is(err, host.ErrNoModel) {
		return State{
			Floor: defaultFloor, Source: SourceDefault, Model: "", AutomaticFloor: defaultFloor,
			IsManual: false, ManualFloor: 0, Calibration: nil,
		}, nil
	}
	if err != nil {
		return State{}, fmt.Errorf("read the embedding state: %w", err)
	}
	return f.getState(ctx, s, state)
}

func (f *Floors) getState(ctx context.Context, s scope, embedding host.EmbeddingState) (State, error) {
	applied, err := s.search.Policy.GetSearchFloor(ctx, embedding.Recipe)
	if err != nil {
		return State{}, fmt.Errorf("read the floor: %w", err)
	}
	record, _, err := s.records.Read(ctx, embedding.Recipe)
	if err != nil {
		return State{}, err
	}
	return State{
		Floor:          applied.Floor,
		Source:         getSource(record, applied),
		Model:          embedding.Model,
		AutomaticFloor: getAutomaticFloor(record, applied),
		IsManual:       record.IsManual,
		ManualFloor:    record.ManualFloor,
		Calibration:    record.Calibration,
	}, nil
}

func getSource(record Record, applied host.SearchFloor) Source {
	switch applied.Source {
	case host.FloorFromStore:
		if record.IsManual {
			return SourceManual
		}
		return SourceCalibrated
	case host.FloorFromModelHint:
		return SourceModel
	case host.FloorFromDefault:
		// gomutants:disable-next-line BRANCH_CASE reason="The switch fallback also returns SourceDefault."
		return SourceDefault
	}
	return SourceDefault
}

func getAutomaticFloor(record Record, applied host.SearchFloor) float64 {
	switch {
	case record.Calibration != nil && record.Calibration.Floor > 0:
		return record.Calibration.Floor
	case applied.Hint > 0:
		return applied.Hint
	default:
		return applied.Default
	}
}

// Calibrate measures the vault and keeps the result. The host applies the measured floor unless the
// person set one. A vault with too few notes is unchanged.
func (f *Floors) Calibrate(ctx context.Context, vault domain.VaultID) (Outcome, State, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := f.open(vault)
	embedding, err := s.search.Index.GetEmbeddingState(ctx)
	if err != nil {
		return 0, State{}, fmt.Errorf("read the embedding state: %w", err)
	}
	outcome, calibration, err := f.calibrator.Calibrate(ctx, s.search.Index, s.notes, embedding)
	if err != nil {
		return 0, State{}, err
	}
	if outcome == Calibrated {
		if err := f.keep(ctx, s, embedding.Recipe, calibration); err != nil {
			return 0, State{}, err
		}
	}
	state, err := f.getState(ctx, s, embedding)
	return outcome, state, err
}

func (f *Floors) keep(ctx context.Context, s scope, recipe string, calibration Calibration) error {
	record, _, err := s.records.Read(ctx, recipe)
	if err != nil {
		return err
	}
	record.Recipe = recipe
	record.Calibration = &calibration
	if err := s.records.Write(ctx, record); err != nil {
		return err
	}
	return apply(ctx, s, record)
}

// Set puts the floor of the person in force, or returns to the automatic floor. A cleared floor is kept as zero.
func (f *Floors) Set(ctx context.Context, vault domain.VaultID, isManual bool, floor float64) (State, error) {
	if isManual && !(floor > 0 && floor <= 1) {
		return State{}, host.ErrInvalidFloor
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	s := f.open(vault)
	embedding, err := s.search.Index.GetEmbeddingState(ctx)
	if err != nil {
		return State{}, fmt.Errorf("read the embedding state: %w", err)
	}
	record, _, err := s.records.Read(ctx, embedding.Recipe)
	if err != nil {
		return State{}, err
	}
	record.Recipe = embedding.Recipe
	record.IsManual = isManual
	record.ManualFloor = 0
	if isManual {
		record.ManualFloor = floor
	}
	if err := s.records.Write(ctx, record); err != nil {
		return State{}, err
	}
	if err := apply(ctx, s, record); err != nil {
		return State{}, err
	}
	return f.getState(ctx, s, embedding)
}

func apply(ctx context.Context, s scope, record Record) error {
	wanted := 0.0
	switch {
	case record.IsManual && record.ManualFloor > 0:
		wanted = record.ManualFloor
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="Assigning a zero floor in the final case leaves wanted at zero."
	case record.Calibration != nil && record.Calibration.Floor > 0:
		wanted = record.Calibration.Floor
	}
	applied, err := s.search.Policy.GetSearchFloor(ctx, record.Recipe)
	if err != nil {
		return fmt.Errorf("read the floor: %w", err)
	}
	if applied.Stored == wanted {
		return nil
	}
	if err := s.search.Policy.SetSearchFloor(ctx, record.Recipe, wanted); err != nil {
		return fmt.Errorf("set the floor: %w", err)
	}
	return nil
}

// Probe embeds the query and measures the nearest passage of the vault. A query of no words is not
// embedded and has no passage near it.
func (f *Floors) Probe(ctx context.Context, vault domain.VaultID, query string) (ProbeResult, error) {
	s := f.open(vault)
	embedding, err := s.search.Index.GetEmbeddingState(ctx)
	if err != nil {
		return ProbeResult{}, fmt.Errorf("read the embedding state: %w", err)
	}
	applied, err := s.search.Policy.GetSearchFloor(ctx, embedding.Recipe)
	if err != nil {
		return ProbeResult{}, fmt.Errorf("read the floor: %w", err)
	}
	if strings.TrimSpace(query) == "" {
		return ProbeResult{BestSimilarity: 0, Floor: applied.Floor, DoesPass: false}, nil
	}
	embedded, err := s.search.Index.EmbedTexts(ctx, []string{clipText(query, host.MaxEmbeddedBytes)})
	if err != nil {
		return ProbeResult{}, fmt.Errorf("embed the query: %w", err)
	}
	if len(embedded.Vectors) != 1 {
		return ProbeResult{}, fmt.Errorf("embed the query: %w: %d vectors for one query", host.ErrEmbeddingFailed, len(embedded.Vectors))
	}
	best, err := s.search.Index.GetBestSimilarity(ctx, embedded.Recipe, embedded.Vectors[0])
	if err != nil {
		return ProbeResult{}, fmt.Errorf("read the nearest passage: %w", err)
	}
	return ProbeResult{BestSimilarity: best, Floor: applied.Floor, DoesPass: best >= applied.Floor}, nil
}

// Ensure makes the host apply the floor of the vault, in automatic mode: where the vault is embedded
// and nothing is kept for the model in use, the vault is measured, and where a record exists the host
// is made to apply what it stands for. A vault that is not embedded is left as it is, and so is a vault with too few
// notes to measure, which is measured again at the next call. Calling it again changes nothing that is done.
func (f *Floors) Ensure(ctx context.Context, vault domain.VaultID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := f.open(vault)
	embedding, err := s.search.Index.GetEmbeddingState(ctx)
	if errors.Is(err, host.ErrNoModel) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read the embedding state: %w", err)
	}
	if !embedding.IsReady {
		return nil
	}
	record, found, err := s.records.Read(ctx, embedding.Recipe)
	if err != nil {
		return err
	}
	if found && (record.IsManual || record.Calibration != nil) {
		return apply(ctx, s, record)
	}
	outcome, calibration, err := f.calibrator.Calibrate(ctx, s.search.Index, s.notes, embedding)
	// gomutants:disable-next-line EXPRESSION_REMOVE reason="Failed calibration always returns a zero outcome."
	if err != nil || outcome != Calibrated {
		return err
	}
	return f.keep(ctx, s, embedding.Recipe, calibration)
}
