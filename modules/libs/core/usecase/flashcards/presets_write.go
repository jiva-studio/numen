package flashcards

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

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

// ErrNoPresets is a build carrying no index. It reaches no preset by name, so
// it points no deck at one, and nothing is written.
var ErrNoPresets = errors.New("this build cannot work the presets of a vault")

// Point puts the deck at path on a preset, by writing the entry of its `links:`
// block that carries `type: preset`.
//
// An empty preset takes that entry out, and the deck is scheduled by the
// defaults. A path naming a note that is not a preset is refused ErrNotAPreset,
// and a path the vault holds no note at is refused note.ErrNoNote.
//
// Fingerprint, when it is given, is what the caller believes is on disk. A deck
// that has changed since it was read is left alone and port.ErrChanged comes
// back. What comes back otherwise is the fingerprint of the file this write
// produced, which is what the caller presents at its next write. An index that
// could not be brought level is note.ErrUnlevelled beside that fingerprint.
func (u Presets) Point(
	ctx context.Context, v domain.Vault, deck, preset string, fingerprint domain.FileRef,
) (domain.FileRef, error) {
	var to domain.Address
	if preset != "" {
		if u.Notes == nil {
			return domain.FileRef{}, ErrNoPresets
		}
		at, err := u.Read(ctx, v, preset)
		if err != nil {
			return domain.FileRef{}, err
		}
		switch {
		case at.Outcome == note.Missing:
			return domain.FileRef{}, fmt.Errorf("%w: %s", note.ErrNoNote, preset)
		case at.Outcome != note.Ok || at.Type != domain.TypePreset:
			return domain.FileRef{}, fmt.Errorf("%w: %s", ErrNotAPreset, preset)
		}
		if to, err = note.Addressed(ctx, u.Notes, v.ID, preset); err != nil {
			return domain.FileRef{}, err
		}
	}
	linking := note.Linking{Readers: u.Readers, Writers: u.Writers, Index: u.Index}
	return linking.PointAt(ctx, v, deck, LinkType, to, domain.RoleRef, fingerprint)
}

// Save writes the settings into the preset at path.
//
// Each key the application owns is replaced on its own. Every other key, the
// order they stand in, the way they are written and the body are left as they
// arrived.
//
// Fingerprint, when it is given, is what the caller believes is on disk. A note
// that has changed since it was read is left alone and port.ErrChanged comes
// back. What comes back otherwise is the fingerprint of the file this write
// produced, which is what the caller presents at its next write. An index that
// could not be brought level is note.ErrUnlevelled beside that fingerprint.
func (u Presets) Save(
	ctx context.Context, v domain.Vault, path string, settings history.Preset,
	fingerprint domain.FileRef,
) (domain.FileRef, error) {
	if err := bounded(settings); err != nil {
		return domain.FileRef{}, err
	}
	at, err := u.save(ctx, v, path, settings, fingerprint)
	if err != nil || u.Index == nil {
		return at, err
	}
	return at, note.Levelled(path, u.Index(ctx, v, []string{path}))
}

// save is the read, the change and the write, under this vault's write lock
// from before the read until after the file is replaced.
func (u Presets) save(
	ctx context.Context, v domain.Vault, path string, settings history.Preset,
	fingerprint domain.FileRef,
) (domain.FileRef, error) {
	release, err := u.Writers.Hold(ctx, v)
	if err != nil {
		return domain.FileRef{}, err
	}
	defer release()

	reader, err := u.Readers.Open(v)
	if err != nil {
		return domain.FileRef{}, err
	}
	on, err := reader.Stat(ctx, path)
	if err != nil {
		return domain.FileRef{}, fmt.Errorf("look at %s: %w", path, missing(err))
	}
	// The bound a note is written under is the bound it is read under.
	if on.Size > note.MaxBytes {
		return domain.FileRef{}, fmt.Errorf("%w: %s is %d bytes, and %d is the most",
			note.ErrTooLarge, path, on.Size, note.MaxBytes)
	}
	// A caller that said what it believed the note was is held to that; one that
	// said nothing is held to what stands there now.
	against := fingerprint
	if against == (domain.FileRef{}) {
		against = on
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return domain.FileRef{}, fmt.Errorf("read %s: %w", path, missing(err))
	}
	n := markdown.Parse(against, raw)
	if n.Type != domain.TypePreset {
		return domain.FileRef{}, fmt.Errorf("%w: %s is a %s", ErrNotAPreset, path, n.Type)
	}
	// What the note said before this write, so that a setting the read could not
	// make out is one this write leaves standing.
	was, _ := history.ReadPreset(n.Frontmatter)

	doc, err := markdown.Open(raw)
	if err != nil {
		return domain.FileRef{}, fmt.Errorf("%s: %w", path, err)
	}
	if err := settle(doc, settings, was); err != nil {
		return domain.FileRef{}, fmt.Errorf("%s: %w", path, err)
	}

	writer, err := u.Writers.Open(v)
	if err != nil {
		return domain.FileRef{}, err
	}
	return writer.Write(ctx, path, doc.Bytes(), against)
}

// settle writes the settings into the frontmatter, one key at a time. A day the
// goal does not name and a week of days all carrying the whole load are keys
// the note stops carrying.
//
// Was is the preset the note was read as. A key is written only where the
// setting differs from it, so a key the read could not make out keeps the words
// the person wrote and a save changing one control leaves the rest of the file
// byte for byte. A key the read could not make out and the person has since
// moved is a setting that differs, and is written.
func settle(doc *markdown.Document, p, was history.Preset) error {
	if !p.By.Equal(was.By) {
		if p.By.IsZero() {
			if err := doc.SetScalar(byDateKey, ""); err != nil {
				return err
			}
		} else if err := doc.SetDay(byDateKey, p.By); err != nil {
			return err
		}
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
		{evenLoadKey, p.EvenLoad, was.EvenLoad},
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
	return doc.SetMapping(loadKey, shares(entries, p.Load, was.Load))
}

// shares is the `load` block a save puts down: a day the read made out carries
// what the settings say or is taken out, and every other entry stands where it
// was, in the order it was written in. A day the block does not name is written
// after the ones it does.
//
// A day named more than once, under names that differ only in how they are
// written, is one day to the read. The entry the read took carries the share
// and the rest are taken out.
func shares(entries []string, load, read map[time.Weekday]int) []markdown.Entry {
	out := make([]markdown.Entry, 0, len(entries)+len(load))
	taken := reading(entries, read)
	written := make(map[time.Weekday]bool, len(load))
	for _, name := range entries {
		weekday, isDay := history.Weekday(name)
		_, could := read[weekday]
		if !isDay || !could {
			out = append(out, markdown.Entry{Key: name, Standing: true})
			continue
		}
		share, named := load[weekday]
		if !named || name != taken[weekday] {
			continue
		}
		written[weekday] = true
		out = append(out, markdown.Entry{Key: name, Value: share})
	}
	for _, weekday := range week {
		if share, named := load[weekday]; named && !written[weekday] {
			out = append(out, markdown.Entry{Key: history.DayName(weekday), Value: share})
		}
	}
	return out
}

// reading is the entry a read takes for each day of the week. Keys reach the
// reader in the order their names sort in, so the last of the names standing
// for one day is the one whose share it holds.
func reading(entries []string, read map[time.Weekday]int) map[time.Weekday]string {
	out := make(map[time.Weekday]string, len(entries))
	for _, name := range entries {
		weekday, isDay := history.Weekday(name)
		if _, could := read[weekday]; !isDay || !could {
			continue
		}
		if standing, held := out[weekday]; !held || name > standing {
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

// bounded is what in the settings may not be written, and is nil when all of
// them may.
func bounded(p history.Preset) error {
	if !history.KnownGoal(p.Goal) {
		return fmt.Errorf("%w: goal %s is not %s, %s or %s",
			ErrOutOfBounds, p.Goal, history.GoalMinutes, history.GoalRetention, history.GoalDate)
	}
	if p.Goal == history.GoalDate && p.By.IsZero() {
		return fmt.Errorf("%w: a preset aiming at a day says which day", ErrOutOfBounds)
	}
	if !history.KnownRule(p.Rule) {
		return fmt.Errorf("%w: learned %s is not %s or %s",
			ErrOutOfBounds, p.Rule, history.RuleInterval, history.RuleRetention)
	}
	if !history.KnownCounts(p.Counts) {
		return fmt.Errorf("%w: counts %s is not %s or %s",
			ErrOutOfBounds, p.Counts, history.CountsCards, history.CountsShows)
	}
	for _, one := range []struct {
		key    string
		value  float64
		bounds history.Bounds
	}{
		{minutesADayKey, float64(p.MinutesADay), history.MinutesADayBounds},
		{newADayKey, float64(p.NewADay), history.NewADayBounds},
		{reviewsADayKey, float64(p.ReviewsADay), history.ReviewsADayBounds},
		{retentionKey, p.Retention, history.RetentionBounds},
		{intervalKey, float64(p.Interval), history.IntervalBounds},
		{backlogKey, float64(p.Backlog), history.BacklogBounds},
	} {
		if !one.bounds.Holds(one.value) {
			return fmt.Errorf("%w: %s %g is outside %g to %g",
				ErrOutOfBounds, one.key, one.value, one.bounds.Least, one.bounds.Most)
		}
	}
	for _, weekday := range week {
		share, named := p.Load[weekday]
		if !named {
			continue
		}
		if !history.LoadBounds.Holds(float64(share)) {
			return fmt.Errorf("%w: the load of %s, %d, is outside %g to %g",
				ErrOutOfBounds, history.DayName(weekday), share,
				history.LoadBounds.Least, history.LoadBounds.Most)
		}
	}
	for weekday := range p.Load {
		if weekday < time.Sunday || weekday > time.Saturday {
			return fmt.Errorf("%w: a load is kept on a day of the week", ErrOutOfBounds)
		}
	}
	return nil
}

// missing is note.ErrNoNote where the vault holds nothing at the path, and the
// error as it arrived otherwise.
func missing(err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return note.ErrNoNote
	}
	return err
}
