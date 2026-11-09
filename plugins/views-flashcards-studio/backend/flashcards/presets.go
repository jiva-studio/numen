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

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/markdown"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/note"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

// LinkType is what a deck's link to its preset carries under `type`.
const LinkType = "preset"

// PresetContents is one preset as returned by a read.
type PresetContents struct {
	Path       string
	Outcome    note.ReadOutcome
	Type       domain.NoteType
	Settings   review.Preset
	Stops      review.StopReason
	StopsToday review.StopReason
	Problems   []string
	FileStat   domain.FileStat
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
	notes    ports.NoteQueries
	Problems ports.ProblemQueries
	index    note.IndexUpdater
	day      review.Day
	now      ports.Clock
}

// NewPresets creates the use case through which a vault's presets are read and
// written. It takes the vault readers and writers, the deck-link resolver, the
// note queries that identify presets and their names, the index updater, and the clock.
func NewPresets(
	readers ports.VaultReaders,
	writers ports.VaultWriters,
	links ports.LinkQueries,
	notes ports.NoteQueries,
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

func (u Presets) getStopReasons(p review.Preset) (review.StopReason, review.StopReason) {
	pos := u.now.Now()
	return p.GetOverallStopReason(u.day, pos), p.GetStopReason(u.day, pos)
}

// PresetSummary is one preset as listed for selection: the file path and its
// name. A note without a name has an empty title and is identified by its path.
type PresetSummary struct {
	Path  string
	Title string
}

// List returns every preset the vault holds, by path. The defaults are excluded: they schedule a
// deck that names no preset, and no note holds them. The index identifies the preset notes, so no
// file is opened. A build without the index gets ErrNoPresets, because it must not answer with an
// empty list it cannot determine.
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
	return PresetContents{Path: "", Outcome: note.Ok, Type: "", Settings: review.Defaults(), Stops: "", StopsToday: "", Problems: nil, FileStat: domain.FileStat{Path: "", Kind: "", Size: 0, ModTime: time.Time{}}}
}

// GetForDeck is the preset the deck at path is scheduled by.
// A link reaching nothing, a link reaching a note that is not a preset, and a `links:` entry with
// no role all leave the deck on the defaults and say so against it. A deck naming two presets is
// scheduled by the first and carries a problem.
func (u Presets) GetForDeck(ctx context.Context, v domain.Vault, deck string) (PresetContents, error) {
	return u.OpenReads().GetForDeck(ctx, v, deck)
}

// PresetReads is a run of reads over one vault. It caches each preset note it
// opens and each deck it resolves for the duration of the run.
//
// A PresetReads is scoped to one call: a preset read from it reflects the file
// as it was when the run began.
type PresetReads struct {
	Presets
	cachedPresets map[string]PresetContents
	scheduling    map[string]PresetContents
	parseProblems map[string][]string
}

// OpenReads opens a run of reads sharing the notes they open.
func (u Presets) OpenReads() *PresetReads {
	return &PresetReads{
		Presets:       u,
		cachedPresets: make(map[string]PresetContents),
		scheduling:    make(map[string]PresetContents),
		parseProblems: nil,
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

	out, delta, err := r.readFirstLink(ctx, v, matchedLinks[0])
	if err != nil {
		return PresetContents{}, err
	}
	if len(matchedLinks) > 1 {
		out.Problems = append(slices.Clone(out.Problems),
			"this deck names more than one preset, and is scheduled by "+delta)
	}
	return out, nil
}

func (r *PresetReads) readFirstLink(ctx context.Context, v domain.Vault, first domain.ResolvedLink) (PresetContents, string, error) {
	delta := first.Target.GetWritten()
	out := CreateDefaultPreset()
	if first.To == "" {
		out.Problems = []string{delta + " reaches no note, and the defaults stand"}
		return out, delta, nil
	}
	delta = first.To
	out, err := r.read(ctx, v, first.To)
	if err != nil {
		return PresetContents{}, delta, err
	}
	// A note that is not a preset schedules nothing, so the deck is treated like
	// the decks that name no preset and the problem is reported against it.
	if out.Type != format.TypePreset {
		out.Path = ""
	}
	return out, delta, nil
}

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

func (u Presets) readPreset(ctx context.Context, v domain.Vault, path string) (PresetContents, error) {
	out := PresetContents{Path: path, Outcome: "", Type: "", Settings: review.Defaults(), Stops: "", StopsToday: "", Problems: nil, FileStat: domain.FileStat{Path: "", Kind: "", Size: 0, ModTime: time.Time{}}}

	reader, err := u.readers.Open(v)
	if err != nil {
		return PresetContents{}, err
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

	return parsePresetBytes(out, ref, raw)
}

func parsePresetBytes(out PresetContents, ref domain.FileStat, raw []byte) (PresetContents, error) {
	if !utf8.Valid(raw) {
		out.Outcome = note.NotText
		return out, nil
	}
	if _, err := markdown.Open(raw); errors.Is(err, markdown.ErrUnreadable) {
		out.Outcome = note.Unreadable
		return out, nil
		// gomutants:disable-next-line BRANCH_IF reason="Markdown Open returns only nil or ErrUnreadable."
	} else if err != nil {
		return PresetContents{}, fmt.Errorf("read %s: %w", out.Path, err)
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

// Point puts the deck at path on a preset, by writing the entry of its `links:` block that
// carries `type: preset`. An empty preset takes that entry out, and the deck is scheduled by the
// defaults. A deck changed since stat was read is left alone and port.ErrStale is returned. A
// landed write answers with the new file stat; an index that could not be updated is
// note.ErrIndexOutdated beside it.
func (u Presets) Point(
	ctx context.Context, v domain.Vault, deck, preset string, stat domain.FileStat,
) (domain.FileStat, error) {
	to, err := u.getPresetAddress(ctx, v, preset)
	if err != nil {
		return domain.FileStat{}, err
	}
	linking := note.NewEditLinks(u.readers, u.writers, u.index, u.now).WithStat(stat)
	return linking.PointAt(ctx, v, deck, LinkType, to, domain.RoleRef)
}

func (u Presets) getPresetAddress(ctx context.Context, v domain.Vault, preset string) (domain.Address, error) {
	var to domain.Address
	if preset == "" {
		// gomutants:disable-next-line RETURN_ZERO reason="Address variable to is already the zero value at this return."
		return to, nil
	}
	if u.notes == nil {
		// gomutants:disable-next-line RETURN_ZERO reason="Address variable to is already the zero value at this return."
		return to, ErrNoPresets
	}
	pos, err := u.Read(ctx, v, preset)
	if err != nil {
		// gomutants:disable-next-line RETURN_ZERO reason="Address variable to is already the zero value at this return."
		return to, err
	}
	switch {
	case pos.Outcome == note.Missing:
		// gomutants:disable-next-line RETURN_ZERO reason="Address variable to is already the zero value at this return."
		return to, fmt.Errorf("%w: %s", note.ErrNoNote, preset)
	case pos.Outcome != note.Ok || pos.Type != format.TypePreset:
		// gomutants:disable-next-line RETURN_ZERO reason="Address variable to is already the zero value at this return."
		return to, fmt.Errorf("%w: %s", ErrNotAPreset, preset)
	}
	return note.GetAddress(ctx, u.notes, v.ID, preset)
}

// Save writes the settings into the preset at path.
// Each key the application owns is replaced on its own; every other key, its order, its
// formatting and the body are left as they arrived. A note changed since stat was read is left
// alone and port.ErrStale is returned. A landed write answers with the new file stat; an index
// that could not be updated is note.ErrIndexOutdated beside it.
func (u Presets) Save(
	ctx context.Context, v domain.Vault, path string, settings review.Preset,
	stat domain.FileStat,
) (domain.FileStat, error) {
	if err := checkBounds(settings); err != nil {
		return domain.FileStat{}, err
	}
	pos, err := u.save(ctx, v, path, settings, stat)
	if err != nil {
		// gomutants:disable-next-line RETURN_ZERO reason="Save helper returns a zero FileStat alongside any error."
		return pos, err
	}
	return pos, note.WrapIndexOutdated(u.index(ctx, v, []string{path}), path)
}

func (u Presets) save(
	ctx context.Context, v domain.Vault, path string, settings review.Preset,
	stat domain.FileStat,
) (saved domain.FileStat, err error) {
	release, err := u.writers.Lock(ctx, v)
	if err != nil {
		return domain.FileStat{}, err
	}
	defer func() { err = errors.Join(err, release()) }()

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
	// The note's preset before this write, so a setting the read could not
	// parse is left unchanged.
	was, _ := review.ReadPreset(n.Frontmatter)

	doc, err := markdown.Open(raw)
	// gomutants:disable-next-line BRANCH_IF reason="Notes with parsed preset frontmatter always open as valid markdown documents."
	if err != nil {
		return domain.FileStat{}, fmt.Errorf("%s: %w", path, err)
	}
	// gomutants:disable-next-line BRANCH_IF reason="Setting scalar and mapping values on a parsed preset document does not fail."
	if err := writeSettings(doc, settings, was); err != nil {
		return domain.FileStat{}, fmt.Errorf("%s: %w", path, err)
	}

	writer, err := u.writers.Open(v)
	if err != nil {
		return domain.FileStat{}, err
	}
	return writer.Write(ctx, path, doc.Bytes(), expectedFp)
}

func writeSettings(doc *markdown.Document, p, was review.Preset) error {
	// gomutants:disable-next-line BRANCH_IF reason="Setting by_date on a parsed preset document does not fail."
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
		// gomutants:disable-next-line BRANCH_IF reason="Setting primitive frontmatter values on a parsed preset document does not fail."
		if err := doc.SetValue(one.key, one.value); err != nil {
			return err
		}
	}
	if maps.Equal(p.Load, was.Load) {
		return nil
	}
	entries, mapping := doc.EntryNames(loadKey)
	// A `load` written in any form other than a week of days belongs to the user
	// and is kept unchanged.
	if !mapping {
		return nil
	}
	return doc.SetMapping(loadKey, readShares(entries, p.Load, was.Load))
}

func writeByDate(doc *markdown.Document, p, was review.Preset) error {
	if p.By.Equal(was.By) {
		return nil
	}
	if p.By.IsZero() {
		return doc.SetScalar(byDateKey, "")
	}
	return doc.SetDay(byDateKey, p.By)
}

func readShares(entries []string, load, read map[time.Weekday]int) []markdown.Entry {
	out := make([]markdown.Entry, 0, len(entries)+len(load))
	dayEntries := getDayEntries(entries, read)
	content := make(map[time.Weekday]bool, len(load))
	for _, name := range entries {
		weekday, isDay := review.Weekday(name)
		_, could := read[weekday]
		if !isDay || !could {
			out = append(out, markdown.Entry{Key: name, Value: nil, IsVerbatim: true})
			continue
		}
		share, named := load[weekday]
		if !named || name != dayEntries[weekday] {
			continue
		}
		content[weekday] = true
		out = append(out, markdown.Entry{Key: name, Value: share, IsVerbatim: false})
	}
	for _, weekday := range getWeek() {
		if share, named := load[weekday]; named && !content[weekday] {
			out = append(out, markdown.Entry{Key: review.DayName(weekday), Value: share, IsVerbatim: false})
		}
	}
	return out
}

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

func getWeek() []time.Weekday {
	return []time.Weekday{
		time.Monday, time.Tuesday, time.Wednesday, time.Thursday,
		time.Friday, time.Saturday, time.Sunday,
	}
}

func checkBounds(p review.Preset) error {
	if !review.IsKnownGoal(p.Goal) {
		return fmt.Errorf("%w: goal %s is not %s, %s or %s",
			ErrOutOfBounds, p.Goal, review.GoalMinutes, review.GoalRetention, review.GoalDate)
	}
	if p.Goal == review.GoalDate && p.By.IsZero() {
		return fmt.Errorf("%w: a preset targeting a day requires the day", ErrOutOfBounds)
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
		{minutesADayKey, float64(p.MinutesADay), review.GetMinutesADayBounds()},
		{newADayKey, float64(p.NewADay), review.GetNewADayBounds()},
		{reviewsADayKey, float64(p.ReviewsADay), review.GetReviewsADayBounds()},
		{retentionKey, p.Retention, review.GetRetentionBounds()},
		{intervalKey, float64(p.Interval), review.GetIntervalBounds()},
		{backlogKey, float64(p.Backlog), review.GetBacklogBounds()},
	} {
		if !one.bounds.Contains(one.value) {
			return fmt.Errorf("%w: %s %g is outside %g to %g",
				ErrOutOfBounds, one.key, one.value, one.bounds.Min, one.bounds.Max)
		}
	}
	return checkLoad(p.Load)
}

func checkLoad(load map[time.Weekday]int) error {
	for _, weekday := range getWeek() {
		share, named := load[weekday]
		if named && !review.GetLoadBounds().Contains(float64(share)) {
			return fmt.Errorf("%w: the load of %s, %d, is outside %g to %g",
				ErrOutOfBounds, review.DayName(weekday), share,
				review.GetLoadBounds().Min, review.GetLoadBounds().Max)
		}
	}
	for weekday := range load {
		if weekday < time.Sunday || weekday > time.Saturday {
			return fmt.Errorf("%w: a load is kept on a day of the week", ErrOutOfBounds)
		}
	}
	return nil
}

func mapMissingNote(err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return note.ErrNoNote
	}
	return err
}
