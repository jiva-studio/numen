package flashcards

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// LinkType is what a deck's link to its preset carries under `type`.
const LinkType = "preset"

// PresetContents is one preset as a read hands it over.
type PresetContents struct {
	// Path is the note the settings were read from, and is empty for a deck
	// naming no preset.
	Path    string
	Outcome note.ReadOutcome
	// Type is what the note at the path says it is, so a caller handed an
	// ordinary note is told so.
	Type domain.NoteType
	// Settings are how the decks pointing here are scheduled. They stand at the
	// defaults for every outcome but Ok.
	Settings review.Preset
	// Stops is why the preset schedules nothing, and empty where it schedules
	// something. It is a fact about the preset and holds on every day.
	Stops review.StopReason
	// StopsToday is why it schedules nothing on the day this was read in. A
	// preset that schedules is stopped today by a day of the week carrying none
	// of the load.
	StopsToday review.StopReason
	// Problems are what was wrong in the file and was not guessed at. They are
	// shown against the preset, and the editor is where they are settled.
	Problems    []string
	Fingerprint domain.Fingerprint
}

// Presets is how each deck of a vault is scheduled.
//
// A deck names its preset with an entry of its `links:` block carrying
// `type: preset`. A deck naming none is scheduled by the defaults, which is
// what a vault holding no preset at all gets.
type Presets struct {
	Readers port.VaultReaders
	Writers port.VaultWriters
	// Links answers where a deck's link to its preset lands.
	Links port.LinkQueries
	// Notes says which notes of the vault are presets and what each is called,
	// and answers what name a link written to one reaches. A build holding none
	// lists no preset and points no deck at one.
	Notes port.NoteQueries
	// Problems is what parsing each file of the vault turned up. A build holding
	// none says nothing against a deck whose link the parser could not read.
	Problems port.ProblemQueries
	// Index brings what a write touched up to date. Every write here calls it,
	// and a preset short of it cannot point a deck at anything.
	Index note.Levels
	// Day is where one day of review gives way to the next, and Now what time
	// it is. They answer whether a preset schedules anything today.
	Day review.Day
	Now port.Clock
}

// NewPresets is what a vault's presets are read and written through: the vault
// their notes are read out of and written back to, where a deck's link to its
// preset lands, what says which notes are presets and what each is called, what
// brings a write level in the index, where one day of review gives way to the
// next, and what time it is.
//
// All seven are named here because a preset short of any one of them schedules
// a deck by something other than the note a person pointed it at, and says
// nothing about having done so.
func NewPresets(
	readers port.VaultReaders,
	writers port.VaultWriters,
	links port.LinkQueries,
	notes port.NoteQueries,
	index note.Levels,
	day review.Day,
	now port.Clock,
) Presets {
	return Presets{
		Readers: readers, Writers: writers, Links: links, Notes: notes,
		Index: index, Day: day, Now: now,
	}
}

// stops is why a preset schedules nothing, and why it schedules nothing today.
func (u Presets) stops(p review.Preset) (review.StopReason, review.StopReason) {
	at := u.Now()
	return p.Stops(u.Day, at), p.StopsOn(u.Day, at)
}

// PresetSummary is one preset as a person choosing between them sees it: where
// the file is, and what it is called. A note nothing names is a title of
// nothing, and the path says which file it is.
type PresetSummary struct {
	Path  string
	Title string
}

// List is every preset the vault holds, by path. The defaults are not among
// them: they are what schedules a deck naming no preset, and no note holds
// them.
//
// The index says which notes are presets, so no file is opened. A build that
// cannot ask it gets ErrNoPresets: a vault holding none is an answer, and a
// build that cannot tell is not entitled to give it.
func (u Presets) List(ctx context.Context, v domain.Vault) ([]PresetSummary, error) {
	if u.Notes == nil {
		return nil, ErrNoPresets
	}
	paths, err := u.Notes.OfType(ctx, v.ID, domain.TypePreset)
	if err != nil {
		return nil, fmt.Errorf("the presets of %s: %w", v.ID, err)
	}
	if len(paths) == 0 {
		return nil, nil
	}
	titles, err := u.Notes.Notes(ctx, v.ID, paths)
	if err != nil {
		return nil, fmt.Errorf("what the presets of %s are called: %w", v.ID, err)
	}

	out := make([]PresetSummary, 0, len(paths))
	for _, path := range paths {
		out = append(out, PresetSummary{Path: path, Title: titles[path].Title})
	}
	return out, nil
}

// Default is a deck scheduled by no preset.
func Default() PresetContents {
	return PresetContents{Outcome: note.Ok, Settings: review.Defaults()}
}

// Of is the preset the deck at path is scheduled by.
//
// A link reaching nothing, a link reaching a note that is not a preset, and a
// `links:` entry written with no role all leave the deck on the defaults and say
// so against it. A deck naming two presets is scheduled by the first and carries
// a problem: two presets are two answers to one question.
func (u Presets) Of(ctx context.Context, v domain.Vault, deck string) (PresetContents, error) {
	return u.Reading().Of(ctx, v, deck)
}

// PresetReads is a run of reads over one vault, holding each preset note it
// opens and each deck it answers for as long as the run lasts.
//
// It is one call's, and a caller keeps it no longer: a preset read from it is
// the file as it stood when the run began.
type PresetReads struct {
	Presets
	held map[string]PresetContents
	// scheduling is the preset each deck asked about is scheduled by, so a deck
	// is asked once however many times the run comes round to it.
	scheduling map[string]PresetContents
	// said is what parsing turned up against each file, read once for the run
	// and only where a deck names no preset.
	said map[string][]string
}

// Reading opens a run of reads sharing the notes they open.
func (u Presets) Reading() *PresetReads {
	return &PresetReads{
		Presets:    u,
		held:       make(map[string]PresetContents),
		scheduling: make(map[string]PresetContents),
	}
}

// Of is the preset the deck at path is scheduled by.
func (r *PresetReads) Of(ctx context.Context, v domain.Vault, deck string) (PresetContents, error) {
	if held, ok := r.scheduling[deck]; ok {
		return held, nil
	}
	out, err := r.scheduled(ctx, v, deck)
	if err != nil {
		return PresetContents{}, err
	}
	out.Stops, out.StopsToday = r.stops(out.Settings)
	r.scheduling[deck] = out
	return out, nil
}

// scheduled works out which preset schedules the deck at path.
func (r *PresetReads) scheduled(
	ctx context.Context, v domain.Vault, deck string,
) (PresetContents, error) {
	if r.Links == nil {
		return Default(), nil
	}
	links, err := r.Links.Links(ctx, v.ID, deck)
	if err != nil {
		return PresetContents{}, fmt.Errorf("the links of %s: %w", deck, err)
	}

	var at []domain.ResolvedLink
	for _, link := range links {
		if link.Type == LinkType {
			at = append(at, link)
		}
	}
	if len(at) == 0 {
		out := Default()
		if out.Problems, err = r.roleless(ctx, v, deck); err != nil {
			return PresetContents{}, err
		}
		return out, nil
	}

	first := at[0]
	by := first.Target.Written()
	out := Default()
	if first.To != "" {
		by = first.To
		if out, err = r.read(ctx, v, first.To); err != nil {
			return PresetContents{}, err
		}
		// A note that is not a preset schedules nothing, so the deck stands
		// with the decks naming none and the problem is shown against it.
		if out.Type != domain.TypePreset {
			out.Path = ""
		}
	} else {
		out.Problems = []string{by + " reaches no note, and the defaults stand"}
	}
	if len(at) > 1 {
		out.Problems = append(slices.Clone(out.Problems),
			"this deck names more than one preset, and is scheduled by "+by)
	}
	return out, nil
}

// roleless is what is shown against a deck that names no preset: every entry of
// its `links:` block the parser could not read for want of a role.
//
// Such an entry is not a link, so a preset written in one schedules nothing.
func (r *PresetReads) roleless(ctx context.Context, v domain.Vault, deck string) ([]string, error) {
	if r.Problems == nil {
		return nil, nil
	}
	if r.said == nil {
		noted, err := r.Problems.Noted(ctx, v.ID)
		if err != nil {
			return nil, fmt.Errorf("what was noted in %s: %w", v.ID, err)
		}
		r.said = make(map[string][]string, len(noted))
		for _, one := range noted {
			r.said[one.Path] = append(r.said[one.Path], one.Detail)
		}
	}

	var out []string
	for _, detail := range r.said[deck] {
		if strings.HasSuffix(detail, markdown.NoRole) {
			out = append(out, detail+", so it names no preset")
		}
	}
	return out, nil
}

// read is the preset at path, opened once however many decks name it.
func (r *PresetReads) read(ctx context.Context, v domain.Vault, path string) (PresetContents, error) {
	if held, ok := r.held[path]; ok {
		return held, nil
	}
	out, err := r.Presets.Read(ctx, v, path)
	if err != nil {
		return PresetContents{}, err
	}
	r.held[path] = out
	return out, nil
}

// Read reads the preset at path.
//
// An error is the vault being out of reach. What is wrong with the note itself
// is an outcome or a problem, and the preset stands at the defaults.
func (u Presets) Read(ctx context.Context, v domain.Vault, path string) (PresetContents, error) {
	out, err := u.opened(ctx, v, path)
	if err != nil {
		return PresetContents{}, err
	}
	out.Stops, out.StopsToday = u.stops(out.Settings)
	return out, nil
}

// opened is the note at path as a preset, before it is asked what it schedules.
func (u Presets) opened(ctx context.Context, v domain.Vault, path string) (PresetContents, error) {
	out := PresetContents{Path: path, Settings: review.Defaults()}

	reader, err := u.Readers.Open(v)
	if err != nil {
		return PresetContents{}, err
	}

	// The vault says what is at a path without opening it: a note, a file it
	// leaves alone, or nothing.
	ref, err := reader.Stat(ctx, path)
	held := err == nil
	switch {
	case held:
		out.Fingerprint = ref
		if ref.Kind != domain.KindNote {
			out.Outcome = note.NotANote
			return out, nil
		}
		if ref.Size > note.MaxBytes {
			out.Outcome = note.TooLarge
			return out, nil
		}
	case errors.Is(err, port.ErrNotANote):
		out.Outcome = note.NotANote
		return out, nil
	case !errors.Is(err, fs.ErrNotExist):
		return PresetContents{}, fmt.Errorf("look at %s: %w", path, err)
	}

	raw, err := reader.Read(ctx, path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		out.Outcome = note.Missing
		return out, nil
	case !held && err == nil:
		out.Outcome = note.NotANote
		return out, nil
	case err != nil:
		return PresetContents{}, fmt.Errorf("read %s: %w", path, err)
	}

	if !utf8.Valid(raw) {
		out.Outcome = note.NotText
		return out, nil
	}
	if _, err := markdown.Open(raw); err != nil {
		out.Outcome = note.Unreadable
		return out, nil
	}

	n := markdown.Parse(ref, raw)
	out.Outcome, out.Type = note.Ok, n.Type
	if n.Type != domain.TypePreset {
		out.Problems = append(out.Problems, path+" is not a preset, and the defaults stand")
		return out, nil
	}
	out.Settings, out.Problems = review.ReadPreset(n.Frontmatter)
	return out, nil
}
