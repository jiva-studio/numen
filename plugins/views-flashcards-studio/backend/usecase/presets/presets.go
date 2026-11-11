// Package presets reads and writes the preset notes that decide how each deck of a vault is
// scheduled, and the `links:` entry of a deck that names its preset.
//
// A preset is a note of `type: preset` whose frontmatter holds the settings of
// the review package. A write replaces the keys the application owns and leaves
// every other key, the formatting and the body as they were.
package presets

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/markdown"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/note"
)

// LinkType is what a deck's link to its preset carries under `type`.
const LinkType = "preset"

// Contents is one preset as returned by a read.
type Contents struct {
	Path       string
	Outcome    note.ReadOutcome
	Type       domain.NoteType
	Settings   review.Preset
	Stops      review.StopReason
	StopsToday review.StopReason
	Problems   []string
	FileStat   domain.FileStat
}

// presetNotes is what the presets ask the index: which notes are presets, what they are called, and
// what a link to one is written by.
type presetNotes interface {
	ports.TypePathQueries
	ports.NoteRefQueries
	ports.NameQueries
}

// Presets is how each deck of a vault is scheduled.
//
// A deck names its preset with an entry of its `links:` block carrying
// `type: preset`. A deck naming none is scheduled by the defaults, which is
// what a vault holding no preset at all gets.
type Presets struct {
	readers  ports.VaultReaders
	writers  ports.VaultWriters
	links    ports.LinkQueries
	notes    presetNotes
	Problems ports.ProblemQueries
	index    note.IndexUpdater
	day      review.Day
	now      ports.Clock
}

// New creates the use case through which a vault's presets are read and
// written. It takes the vault readers and writers, the deck-link resolver, the
// note queries that identify presets and their names, the index updater, and the clock.
func New(
	readers ports.VaultReaders,
	writers ports.VaultWriters,
	links ports.LinkQueries,
	notes presetNotes,
	index note.IndexUpdater,
	now ports.Clock,
) Presets {
	return Presets{
		readers: readers, writers: writers, links: links, notes: notes,
		Problems: nil,
		index:    index, day: review.Day{Starts: 0, In: nil}, now: now,
	}
}

// WithDay returns the use case with the review day boundary that decides whether a preset
// schedules anything today.
func (u Presets) WithDay(day review.Day) Presets {
	u.day = day
	return u
}

// HasLinks reports whether the use case can resolve the links of a deck. Without them every deck
// is scheduled by the defaults.
func (u Presets) HasLinks() bool { return u.links != nil }

// GetBacklinks returns the links that point at the note at path.
func (u Presets) GetBacklinks(ctx context.Context, vault domain.VaultID, path string) ([]domain.ResolvedLink, error) {
	return u.links.GetBacklinks(ctx, vault, path)
}

func (u Presets) getStopReasons(p review.Preset) (review.StopReason, review.StopReason) {
	pos := u.now.Now()
	return p.GetOverallStopReason(u.day, pos), p.GetStopReason(u.day, pos)
}

// Summary is one preset as listed for selection: the file path and its
// name. A note without a name has an empty title and is identified by its path.
type Summary struct {
	Path  string
	Title string
}

// List returns every preset the vault holds, by path. The defaults are excluded: they schedule a
// deck that names no preset, and no note holds them. The index identifies the preset notes, so no
// file is opened. A build without the index gets ErrNoPresets, because it must not answer with an
// empty list it cannot determine.
func (u Presets) List(ctx context.Context, v domain.Vault) ([]Summary, error) {
	if u.notes == nil {
		return nil, ErrNoPresets
	}
	paths, err := u.notes.GetPathsOfType(ctx, v.ID, format.TypePreset)
	if err != nil {
		return nil, fmt.Errorf("the presets of %s: %w", v.ID, err)
	}
	if len(paths) == 0 {
		return nil, nil
	}
	titles, err := u.notes.GetNotes(ctx, v.ID, paths)
	if err != nil {
		return nil, fmt.Errorf("what the presets of %s are called: %w", v.ID, err)
	}

	out := make([]Summary, 0, len(paths))
	for _, path := range paths {
		out = append(out, Summary{Path: path, Title: titles[path].Title})
	}
	return out, nil
}

// CreateDefault returns the preset of a deck scheduled by no preset.
func CreateDefault() Contents {
	return Contents{
		Path: "", Outcome: note.Ok, Type: "", Settings: review.Defaults(), Stops: "", StopsToday: "", Problems: nil,
		FileStat: domain.FileStat{Path: "", Kind: "", Size: 0, ModTime: time.Time{}},
	}
}

// GetForDeck is the preset the deck at path is scheduled by.
// A link reaching nothing, a link reaching a note that is not a preset, and a `links:` entry with
// no role all leave the deck on the defaults and say so against it. A deck naming two presets is
// scheduled by the first and carries a problem.
func (u Presets) GetForDeck(ctx context.Context, v domain.Vault, deck string) (Contents, error) {
	return u.OpenReads().GetForDeck(ctx, v, deck)
}

// Reads is a run of reads over one vault. It caches each preset note it
// opens and each deck it resolves for the duration of the run.
//
// A Reads is scoped to one call: a preset read from it reflects the file
// as it was when the run began.
type Reads struct {
	Presets
	cachedPresets map[string]Contents
	scheduling    map[string]Contents
	parseProblems map[string][]string
}

// OpenReads opens a run of reads sharing the notes they open.
func (u Presets) OpenReads() *Reads {
	return &Reads{
		Presets:       u,
		cachedPresets: make(map[string]Contents),
		scheduling:    make(map[string]Contents),
		parseProblems: nil,
	}
}

// GetForDeck is the preset the deck at path is scheduled by.
func (r *Reads) GetForDeck(ctx context.Context, v domain.Vault, deck string) (Contents, error) {
	if target, ok := r.scheduling[deck]; ok {
		return target, nil
	}
	out, err := r.getSchedulingPreset(ctx, v, deck)
	if err != nil {
		return Contents{}, err
	}
	out.Stops, out.StopsToday = r.getStopReasons(out.Settings)
	r.scheduling[deck] = out
	return out, nil
}

func (r *Reads) getSchedulingPreset(
	ctx context.Context, v domain.Vault, deck string,
) (Contents, error) {
	if r.links == nil {
		return CreateDefault(), nil
	}
	links, err := r.links.GetLinks(ctx, v.ID, deck)
	if err != nil {
		return Contents{}, fmt.Errorf("the links of %s: %w", deck, err)
	}

	var matchedLinks []domain.ResolvedLink
	for _, link := range links {
		if link.Type == LinkType {
			matchedLinks = append(matchedLinks, link)
		}
	}
	if len(matchedLinks) == 0 {
		out := CreateDefault()
		if out.Problems, err = r.readRolelessProblems(ctx, v, deck); err != nil {
			return Contents{}, err
		}
		return out, nil
	}

	out, delta, err := r.readFirstLink(ctx, v, matchedLinks[0])
	if err != nil {
		return Contents{}, err
	}
	if len(matchedLinks) > 1 {
		out.Problems = append(slices.Clone(out.Problems),
			"this deck names more than one preset, and is scheduled by "+delta)
	}
	return out, nil
}

func (r *Reads) readFirstLink(ctx context.Context, v domain.Vault, first domain.ResolvedLink) (Contents, string, error) {
	delta := first.Target.GetWritten()
	out := CreateDefault()
	if first.To == "" {
		out.Problems = []string{delta + " reaches no note, and the defaults stand"}
		return out, delta, nil
	}
	delta = first.To
	out, err := r.GetPreset(ctx, v, first.To)
	if err != nil {
		return Contents{}, delta, err
	}
	// A note that is not a preset schedules nothing, so the deck is treated like
	// the decks that name no preset and the problem is reported against it.
	if out.Type != format.TypePreset {
		out.Path = ""
	}
	return out, delta, nil
}

func (r *Reads) readRolelessProblems(ctx context.Context, v domain.Vault, deck string) ([]string, error) {
	if r.Problems == nil {
		return nil, nil
	}
	if r.parseProblems == nil {
		noted, err := r.Problems.GetParseProblems(ctx, v.ID)
		if err != nil {
			return nil, fmt.Errorf("what was noted in %s: %w", v.ID, err)
		}
		r.parseProblems = make(map[string][]string, len(noted))
		for _, one := range noted {
			r.parseProblems[one.Path] = append(r.parseProblems[one.Path], one.Detail)
		}
	}

	var out []string
	for _, detail := range r.parseProblems[deck] {
		if strings.HasSuffix(detail, markdown.NoRole) {
			out = append(out, detail+", so it names no preset")
		}
	}
	return out, nil
}

// GetPreset is the preset at path, read once for the run.
func (r *Reads) GetPreset(ctx context.Context, v domain.Vault, path string) (Contents, error) {
	if target, ok := r.cachedPresets[path]; ok {
		return target, nil
	}
	out, err := r.Read(ctx, v, path)
	if err != nil {
		return Contents{}, err
	}
	r.cachedPresets[path] = out
	return out, nil
}

// Read reads the preset at path.
//
// An error is the vault being out of reach. What is wrong with the note itself
func (u Presets) Read(ctx context.Context, v domain.Vault, path string) (Contents, error) {
	out, err := u.readPreset(ctx, v, path)
	if err != nil {
		return Contents{}, err
	}
	out.Stops, out.StopsToday = u.getStopReasons(out.Settings)
	return out, nil
}

func (u Presets) readPreset(ctx context.Context, v domain.Vault, path string) (Contents, error) {
	out := Contents{Path: path, Outcome: "", Type: "", Settings: review.Defaults(), Stops: "", StopsToday: "", Problems: nil, FileStat: domain.FileStat{Path: "", Kind: "", Size: 0, ModTime: time.Time{}}}

	reader, err := u.readers.Open(v)
	if err != nil {
		return Contents{}, err
	}

	// The vault reports what is at a path without opening it: a note, a file that
	// is not a note, or nothing.
	ref, err := reader.StatFile(ctx, path)
	target := err == nil
	switch {
	case target:
		out.FileStat = ref
		if ref.Kind != domain.KindNote {
			out.Outcome = note.NotANote
			return out, nil
		}
		if ref.Size > note.MaxBytes {
			out.Outcome = note.TooLarge
			return out, nil
		}
	case errors.Is(err, ports.ErrNotANote):
		out.Outcome = note.NotANote
		return out, nil
	case !ports.IsNotExist(err):
		return Contents{}, fmt.Errorf("look at %s: %w", path, err)
	}

	raw, err := reader.Read(ctx, path)
	switch {
	case ports.IsNotExist(err):
		out.Outcome = note.Missing
		return out, nil
	case !target && err == nil:
		out.Outcome = note.NotANote
		return out, nil
	case err != nil:
		return Contents{}, fmt.Errorf("read %s: %w", path, err)
	}

	return parsePresetBytes(out, ref, raw)
}

func parsePresetBytes(out Contents, ref domain.FileStat, raw []byte) (Contents, error) {
	if !utf8.Valid(raw) {
		out.Outcome = note.NotText
		return out, nil
	}
	if _, err := markdown.Open(raw); errors.Is(err, markdown.ErrUnreadable) {
		out.Outcome = note.Unreadable
		return out, nil
		// gomutants:disable-next-line BRANCH_IF reason="Markdown Open returns only nil or ErrUnreadable."
	} else if err != nil {
		return Contents{}, fmt.Errorf("read %s: %w", out.Path, err)
	}

	n := markdown.Parse(ref, raw)
	out.Outcome, out.Type = note.Ok, n.Type
	if n.Type != format.TypePreset {
		out.Problems = append(out.Problems, out.Path+" is not a preset, and the defaults stand")
		return out, nil
	}
	out.Settings, out.Problems = review.ReadPreset(n.Frontmatter)
	return out, nil
}

// The frontmatter keys of a preset's settings. A key outside this list belongs
// to the user and is never written.
const (
	goalKey        = "goal"
	byDateKey      = "by_date"
	MinutesADayKey = "minutes_a_day"
	newADayKey     = "new_a_day"
	reviewsADayKey = "reviews_a_day"
	RetentionKey   = "retention"
	learnedKey     = "learned"
	intervalKey    = "interval"
	countsKey      = "counts"
	backlogKey     = "backlog"
	loadKey        = "load"
	evenLoadKey    = "even_load"
)

// ErrNotAPreset is returned for a note whose `type` is not a preset. Nothing is
// written.
var ErrNotAPreset = errors.New("this note is not a preset")

// ErrOutOfBounds is a setting outside what it may be. Nothing is written, and
// the settings are weighed before the file is opened.
var ErrOutOfBounds = errors.New("this setting is outside what a preset may hold")

// ErrNoPresets is returned by a build that cannot determine which notes of a
// vault are presets. It neither lists presets nor points a deck at one, and
// nothing is written.
var ErrNoPresets = errors.New("this build cannot work the presets of a vault")
