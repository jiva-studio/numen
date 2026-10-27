package flashcards

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/markdown"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/note"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
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
	Problems []string
	FileStat domain.FileStat
}

// Presets is how each deck of a vault is scheduled.
//
// A deck names its preset with an entry of its `links:` block carrying
// `type: preset`. A deck naming none is scheduled by the defaults, which is
// what a vault holding no preset at all gets.
type Presets struct {
	readers ports.VaultReaders
	writers ports.VaultWriters
	// links answers where a deck's link to its preset lands.
	links ports.LinkQueries
	// notes says which notes of the vault are presets and what each is called,
	// and answers what name a link written to one reaches. A build holding none
	// lists no preset and points no deck at one.
	notes ports.NoteQueries
	// Problems is what parsing each file of the vault turned up. A build holding
	// none says nothing against a deck whose link the parser could not read.
	Problems ports.ProblemQueries
	// index brings what a write touched up to date. Every write here calls it,
	// and a preset short of it cannot point a deck at anything.
	index note.Levels
	// day is where one day of review gives way to the next, and now what time
	// it is. They answer whether a preset schedules anything today.
	day review.Day
	now ports.Clock
}

// NewPresets is what a vault's presets are read and written through: the vault
// their notes are read out of and written back to, where a deck's link to its
// preset lands, what says which notes are presets and what each is called, what
// brings a write level in the index, where one day of review gives way to the
// next, and what time it is.
func NewPresets(
	readers ports.VaultReaders,
	writers ports.VaultWriters,
	links ports.LinkQueries,
	notes ports.NoteQueries,
	index note.Levels,
	day review.Day,
	now ports.Clock,
) Presets {
	return Presets{
		readers: readers, writers: writers, links: links, notes: notes,
		index: index, day: day, now: now,
	}
}

// getStopReasons is why a preset schedules nothing, and why it schedules
// nothing today.
func (u Presets) getStopReasons(p review.Preset) (review.StopReason, review.StopReason) {
	pos := u.now()
	return p.GetOverallStopReason(u.day, pos), p.GetStopReason(u.day, pos)
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

	out := make([]PresetSummary, 0, len(paths))
	for _, path := range paths {
		out = append(out, PresetSummary{Path: path, Title: titles[path].Title})
	}
	return out, nil
}

// CreateDefaultPreset returns the preset of a deck scheduled by no preset.
func CreateDefaultPreset() PresetContents {
	return PresetContents{Outcome: note.Ok, Settings: review.Defaults()}
}

// GetForDeck is the preset the deck at path is scheduled by.
//
// A link reaching nothing, a link reaching a note that is not a preset, and a
// `links:` entry written with no role all leave the deck on the defaults and say
// so against it. A deck naming two presets is scheduled by the first and carries
// a problem: two presets are two answers to one question.
func (u Presets) GetForDeck(ctx context.Context, v domain.Vault, deck string) (PresetContents, error) {
	return u.OpenReads().GetForDeck(ctx, v, deck)
}

// PresetReads is a run of reads over one vault, holding each preset note it
// opens and each deck it answers for as long as the run lasts.
//
// It is one call's, and a caller keeps it no longer: a preset read from it is
// the file as it stood when the run began.
type PresetReads struct {
	Presets
	cachedPresets map[string]PresetContents
	// scheduling is the preset each deck asked about is scheduled by, so a deck
	// is asked once however many times the run comes round to it.
	scheduling map[string]PresetContents
	// parseProblems is what parsing turned up against each file, read once for the run
	// and only where a deck names no preset.
	parseProblems map[string][]string
}

// OpenReads opens a run of reads sharing the notes they open.
func (u Presets) OpenReads() *PresetReads {
	return &PresetReads{
		Presets:       u,
		cachedPresets: make(map[string]PresetContents),
		scheduling:    make(map[string]PresetContents),
	}
}

// GetForDeck is the preset the deck at path is scheduled by.
func (r *PresetReads) GetForDeck(ctx context.Context, v domain.Vault, deck string) (PresetContents, error) {
	if target, ok := r.scheduling[deck]; ok {
		return target, nil
	}
	out, err := r.getSchedulingPreset(ctx, v, deck)
	if err != nil {
		return PresetContents{}, err
	}
	out.Stops, out.StopsToday = r.getStopReasons(out.Settings)
	r.scheduling[deck] = out
	return out, nil
}

// getSchedulingPreset works out which preset schedules the deck at path.
func (r *PresetReads) getSchedulingPreset(
	ctx context.Context, v domain.Vault, deck string,
) (PresetContents, error) {
	if r.links == nil {
		return CreateDefaultPreset(), nil
	}
	links, err := r.links.GetLinks(ctx, v.ID, deck)
	if err != nil {
		return PresetContents{}, fmt.Errorf("the links of %s: %w", deck, err)
	}

	var matchedLinks []domain.ResolvedLink
	for _, link := range links {
		if link.Type == LinkType {
			matchedLinks = append(matchedLinks, link)
		}
	}
	if len(matchedLinks) == 0 {
		out := CreateDefaultPreset()
		if out.Problems, err = r.readRolelessProblems(ctx, v, deck); err != nil {
			return PresetContents{}, err
		}
		return out, nil
	}

	first := matchedLinks[0]
	delta := first.Target.GetWritten()
	out := CreateDefaultPreset()
	if first.To != "" {
		delta = first.To
		if out, err = r.read(ctx, v, first.To); err != nil {
			return PresetContents{}, err
		}
		// A note that is not a preset schedules nothing, so the deck stands
		// with the decks naming none and the problem is shown against it.
		if out.Type != format.TypePreset {
			out.Path = ""
		}
	} else {
		out.Problems = []string{delta + " reaches no note, and the defaults stand"}
	}
	if len(matchedLinks) > 1 {
		out.Problems = append(slices.Clone(out.Problems),
			"this deck names more than one preset, and is scheduled by "+delta)
	}
	return out, nil
}

// readRolelessProblems returns what is shown against a deck that names no preset: every entry of
// its `links:` block the parser could not read for want of a role.
//
// Such an entry is not a link, so a preset written in one schedules nothing.
func (r *PresetReads) readRolelessProblems(ctx context.Context, v domain.Vault, deck string) ([]string, error) {
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

// read is the preset at path, opened once however many decks name it.
func (r *PresetReads) read(ctx context.Context, v domain.Vault, path string) (PresetContents, error) {
	if target, ok := r.cachedPresets[path]; ok {
		return target, nil
	}
	out, err := r.Read(ctx, v, path)
	if err != nil {
		return PresetContents{}, err
	}
	r.cachedPresets[path] = out
	return out, nil
}

// Read reads the preset at path.
//
// An error is the vault being out of reach. What is wrong with the note itself
func (u Presets) Read(ctx context.Context, v domain.Vault, path string) (PresetContents, error) {
	out, err := u.readPreset(ctx, v, path)
	if err != nil {
		return PresetContents{}, err
	}
	out.Stops, out.StopsToday = u.getStopReasons(out.Settings)
	return out, nil
}

// readPreset is the note at path as a preset, before it is asked what it
// schedules.
func (u Presets) readPreset(ctx context.Context, v domain.Vault, path string) (PresetContents, error) {
	out := PresetContents{Path: path, Settings: review.Defaults()}

	reader, err := u.readers.Open(v)
	if err != nil {
		return PresetContents{}, err
	}

	// The vault says what is at a path without opening it: a note, a file it
	// leaves alone, or nothing.
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
	case !errors.Is(err, fs.ErrNotExist):
		return PresetContents{}, fmt.Errorf("look at %s: %w", path, err)
	}

	raw, err := reader.Read(ctx, path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		out.Outcome = note.Missing
		return out, nil
	case !target && err == nil:
		out.Outcome = note.NotANote
		return out, nil
	case err != nil:
		return PresetContents{}, fmt.Errorf("read %s: %w", path, err)
	}

	if !utf8.Valid(raw) {
		out.Outcome = note.NotText
		return out, nil
	}
	if _, err := markdown.Open(raw); errors.Is(err, markdown.ErrUnreadable) {
		out.Outcome = note.Unreadable
		return out, nil
	} else if err != nil {
		return PresetContents{}, fmt.Errorf("read %s: %w", path, err)
	}

	n := markdown.Parse(ref, raw)
	out.Outcome, out.Type = note.Ok, n.Type
	if n.Type != format.TypePreset {
		out.Problems = append(out.Problems, path+" is not a preset, and the defaults stand")
		return out, nil
	}
	out.Settings, out.Problems = review.ReadPreset(n.Frontmatter)
	return out, nil
}

// The frontmatter keys a preset's settings stand under. A key outside this list
// belongs to the person and is never written.
const (
	goalKey        = "goal"
	byDateKey      = "by_date"
	minutesADayKey = "minutes_a_day"
	newADayKey     = "new_a_day"
	reviewsADayKey = "reviews_a_day"
	retentionKey   = "retention"
	learnedKey     = "learned"
	intervalKey    = "interval"
	countsKey      = "counts"
	backlogKey     = "backlog"
	loadKey        = "load"
	evenLoadKey    = "even_load"
)

// ErrNotAPreset is a note whose `type` says it is something else. Nothing is
// written.
var ErrNotAPreset = errors.New("this note is not a preset")

// ErrOutOfBounds is a setting outside what it may be. Nothing is written, and
// the settings are weighed before the file is opened.
var ErrOutOfBounds = errors.New("this setting is outside what a preset may hold")

// ErrNoPresets is a build that cannot say which notes of a vault are presets.
// It reaches none of them by name, so it neither lists them nor points a deck
// at one, and nothing is written.
var ErrNoPresets = errors.New("this build cannot work the presets of a vault")

// Point puts the deck at path on a preset, by writing the entry of its `links:`
// block that carries `type: preset`.
//
// # An empty preset takes that entry out, and the deck is scheduled by the
//
// FileStat, when it is given, is what the caller believes is on disk. A deck
// that has changed since it was read is left alone and port.ErrStale comes
// back. A write that landed answers with the file stat of the file it
// produced, which the caller presents at its next write. An index that could
// not be brought level is note.ErrUnlevelled beside that file stat.
func (u Presets) Point(
	ctx context.Context, v domain.Vault, deck, preset string, stat domain.FileStat,
) (domain.FileStat, error) {
	var to domain.Address
	if preset != "" {
		if u.notes == nil {
			return domain.FileStat{}, ErrNoPresets
		}
		pos, err := u.Read(ctx, v, preset)
		if err != nil {
			return domain.FileStat{}, err
		}
		switch {
		case pos.Outcome == note.Missing:
			return domain.FileStat{}, fmt.Errorf("%w: %s", note.ErrNoNote, preset)
		case pos.Outcome != note.Ok || pos.Type != format.TypePreset:
			return domain.FileStat{}, fmt.Errorf("%w: %s", ErrNotAPreset, preset)
		}
		if to, err = note.GetAddress(ctx, u.notes, v.ID, preset); err != nil {
			return domain.FileStat{}, err
		}
	}
	linking := note.NewEditLinks(u.readers, u.writers, u.index, u.now)
	return linking.PointAt(ctx, v, deck, LinkType, to, domain.RoleRef, stat)
}

// Save writes the settings into the preset at path.
//
// Each key the application owns is replaced on its own. Every other key, the
// order they stand in, the way they are written and the body are left as they
// arrived.
//
// FileStat, when it is given, is what the caller believes is on disk. A note
// that has changed since it was read is left alone and port.ErrStale comes
// back. A write that landed answers with the file stat of the file it
// produced, which the caller presents at its next write. An index that could
// not be brought level is note.ErrUnlevelled beside that file stat.
func (u Presets) Save(
	ctx context.Context, v domain.Vault, path string, settings review.Preset,
	stat domain.FileStat,
) (domain.FileStat, error) {
	if err := checkBounds(settings); err != nil {
		return domain.FileStat{}, err
	}
	pos, err := u.save(ctx, v, path, settings, stat)
	if err != nil {
		return pos, err
	}
	return pos, note.WrapUnlevelled(u.index(ctx, v, []string{path}), path)
}

// save is the read, the change and the write, under this vault's write lock
// from before the read until after the file is replaced.
func (u Presets) save(
	ctx context.Context, v domain.Vault, path string, settings review.Preset,
	stat domain.FileStat,
) (domain.FileStat, error) {
	release, err := u.writers.Lock(ctx, v)
	if err != nil {
		return domain.FileStat{}, err
	}
	defer release()

	reader, err := u.readers.Open(v)
	if err != nil {
		return domain.FileStat{}, err
	}
	on, err := reader.StatFile(ctx, path)
	if err != nil {
		return domain.FileStat{}, fmt.Errorf("look at %s: %w", path, mapMissingNote(err))
	}
	// The bound a note is written under is the bound it is read under.
	if on.Size > note.MaxBytes {
		return domain.FileStat{}, fmt.Errorf("%w: %s is %d bytes, and %d is the most",
			note.ErrTooLarge, path, on.Size, note.MaxBytes)
	}
	expectedFp := stat
	if expectedFp.IsZero() {
		expectedFp = on
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return domain.FileStat{}, fmt.Errorf("read %s: %w", path, mapMissingNote(err))
	}
	n := markdown.Parse(expectedFp, raw)
	if n.Type != format.TypePreset {
		return domain.FileStat{}, fmt.Errorf("%w: %s is a %s", ErrNotAPreset, path, n.Type)
	}
	// What the note said before this write, so that a setting the read could not
	// make out is one this write leaves standing.
	was, _ := review.ReadPreset(n.Frontmatter)

	doc, err := markdown.Open(raw)
	if err != nil {
		return domain.FileStat{}, fmt.Errorf("%s: %w", path, err)
	}
	if err := writeSettings(doc, settings, was); err != nil {
		return domain.FileStat{}, fmt.Errorf("%s: %w", path, err)
	}

	writer, err := u.writers.Open(v)
	if err != nil {
		return domain.FileStat{}, err
	}
	return writer.Write(ctx, path, doc.Bytes(), expectedFp)
}

// writeSettings writes the settings into the frontmatter, one key at a time. A day the
// goal does not name and a week of days all carrying the whole load are keys
// the note stops carrying.
//
// Was is the preset the note was read as. A key is written only where the
// setting differs from it, so a key the read could not make out keeps the words
// the person wrote and a save changing one control leaves the rest of the file
// byte for byte. A key the read could not make out and the person has since
// moved is a setting that differs, and is written.
func writeSettings(doc *markdown.Document, p, was review.Preset) error {
	if err := writeByDate(doc, p, was); err != nil {
		return err
	}
	for _, one := range []struct {
		key        string
		value, was any
	}{
		{goalKey, string(p.Goal), string(was.Goal)},
		{countsKey, string(p.Counts), string(was.Counts)},
		{learnedKey, string(p.Rule), string(was.Rule)},
		{minutesADayKey, p.MinutesADay, was.MinutesADay},
		{newADayKey, p.NewADay, was.NewADay},
		{reviewsADayKey, p.ReviewsADay, was.ReviewsADay},
		{backlogKey, p.Backlog, was.Backlog},
		{retentionKey, p.Retention, was.Retention},
		{intervalKey, p.Interval, was.Interval},
		{evenLoadKey, p.IsEvenLoad, was.IsEvenLoad},
	} {
		if one.value == one.was {
			continue
		}
		if err := doc.SetValue(one.key, one.value); err != nil {
			return err
		}
	}
	if maps.Equal(p.Load, was.Load) {
		return nil
	}
	entries, mapping := doc.EntryNames(loadKey)
	// A `load` written as anything but a week of days is the person's, whole.
	if !mapping {
		return nil
	}
	return doc.SetMapping(loadKey, readShares(entries, p.Load, was.Load))
}

// writeByDate writes the day the goal aims at when it differs from the one the
// note was read with, and empties the key when the preset names no day.
func writeByDate(doc *markdown.Document, p, was review.Preset) error {
	if p.By.Equal(was.By) {
		return nil
	}
	if p.By.IsZero() {
		return doc.SetScalar(byDateKey, "")
	}
	return doc.SetDay(byDateKey, p.By)
}

// readShares is the `load` block a save puts down: a day the read made out
// carries what the settings say or is taken out, and every other entry stands
// where it was, in the order it was written in. A day the block does not name
// is written after the ones it does.
//
// A day named more than once, under names that differ only in how they are
// written, is one day to the read. The entry the read took carries the share
// and the rest are taken out.
func readShares(entries []string, load, read map[time.Weekday]int) []markdown.Entry {
	out := make([]markdown.Entry, 0, len(entries)+len(load))
	dayEntries := getDayEntries(entries, read)
	content := make(map[time.Weekday]bool, len(load))
	for _, name := range entries {
		weekday, isDay := review.Weekday(name)
		_, could := read[weekday]
		if !isDay || !could {
			out = append(out, markdown.Entry{Key: name, IsVerbatim: true})
			continue
		}
		share, named := load[weekday]
		if !named || name != dayEntries[weekday] {
			continue
		}
		content[weekday] = true
		out = append(out, markdown.Entry{Key: name, Value: share})
	}
	for _, weekday := range week {
		if share, named := load[weekday]; named && !content[weekday] {
			out = append(out, markdown.Entry{Key: review.DayName(weekday), Value: share})
		}
	}
	return out
}

// getDayEntries returns the entry a read takes for each day of the week. Keys reach the
// reader in the order their names sort in, so the last of the names standing
// for one day is the one whose share it holds.
func getDayEntries(entries []string, read map[time.Weekday]int) map[time.Weekday]string {
	out := make(map[time.Weekday]string, len(entries))
	for _, name := range entries {
		weekday, isDay := review.Weekday(name)
		if _, could := read[weekday]; !isDay || !could {
			continue
		}
		if was, ok := out[weekday]; !ok || name > was {
			out[weekday] = name
		}
	}
	return out
}

// week is the days of the week in the order a preset writes them.
var week = []time.Weekday{
	time.Monday, time.Tuesday, time.Wednesday, time.Thursday,
	time.Friday, time.Saturday, time.Sunday,
}

// checkBounds is what in the settings may not be written, and is nil when all
// of them may.
func checkBounds(p review.Preset) error {
	if !review.IsKnownGoal(p.Goal) {
		return fmt.Errorf("%w: goal %s is not %s, %s or %s",
			ErrOutOfBounds, p.Goal, review.GoalMinutes, review.GoalRetention, review.GoalDate)
	}
	if p.Goal == review.GoalDate && p.By.IsZero() {
		return fmt.Errorf("%w: a preset aiming at a day says which day", ErrOutOfBounds)
	}
	if !review.IsKnownRule(p.Rule) {
		return fmt.Errorf("%w: learned %s is not %s or %s",
			ErrOutOfBounds, p.Rule, review.RuleInterval, review.RuleRetention)
	}
	if !review.IsKnownBudgetUnit(p.Counts) {
		return fmt.Errorf("%w: counts %s is not %s or %s",
			ErrOutOfBounds, p.Counts, review.BudgetUnitCards, review.BudgetUnitShows)
	}
	for _, one := range []struct {
		key    string
		value  float64
		bounds review.Bounds
	}{
		{minutesADayKey, float64(p.MinutesADay), review.MinutesADayBounds},
		{newADayKey, float64(p.NewADay), review.NewADayBounds},
		{reviewsADayKey, float64(p.ReviewsADay), review.ReviewsADayBounds},
		{retentionKey, p.Retention, review.RetentionBounds},
		{intervalKey, float64(p.Interval), review.IntervalBounds},
		{backlogKey, float64(p.Backlog), review.BacklogBounds},
	} {
		if !one.bounds.Contains(one.value) {
			return fmt.Errorf("%w: %s %g is outside %g to %g",
				ErrOutOfBounds, one.key, one.value, one.bounds.Least, one.bounds.Most)
		}
	}
	return checkLoad(p.Load)
}

// checkLoad is what in the week's load may not be written, and is nil when all
// of it may.
func checkLoad(load map[time.Weekday]int) error {
	for _, weekday := range week {
		share, named := load[weekday]
		if named && !review.LoadBounds.Contains(float64(share)) {
			return fmt.Errorf("%w: the load of %s, %d, is outside %g to %g",
				ErrOutOfBounds, review.DayName(weekday), share,
				review.LoadBounds.Least, review.LoadBounds.Most)
		}
	}
	for weekday := range load {
		if weekday < time.Sunday || weekday > time.Saturday {
			return fmt.Errorf("%w: a load is kept on a day of the week", ErrOutOfBounds)
		}
	}
	return nil
}

// mapMissingNote is note.ErrNoNote where the vault holds nothing at the path,
// and the error as it arrived otherwise.
func mapMissingNote(err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return note.ErrNoNote
	}
	return err
}
