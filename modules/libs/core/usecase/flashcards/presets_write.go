package flashcards

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
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
	// A caller that said what it believed the note was is held to that; one that
	// said nothing is held to what stands there now.
	against := fingerprint
	if against == (domain.FileRef{}) {
		on, err := reader.Stat(ctx, path)
		if err != nil {
			return domain.FileRef{}, fmt.Errorf("look at %s: %w", path, missing(err))
		}
		against = on
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return domain.FileRef{}, fmt.Errorf("read %s: %w", path, missing(err))
	}
	if kind := markdown.Parse(against, raw).Type; kind != domain.TypePreset {
		return domain.FileRef{}, fmt.Errorf("%w: %s is a %s", ErrNotAPreset, path, kind)
	}

	doc, err := markdown.Open(raw)
	if err != nil {
		return domain.FileRef{}, fmt.Errorf("%s: %w", path, err)
	}
	if err := settle(doc, settings); err != nil {
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
func settle(doc *markdown.Document, p history.Preset) error {
	if err := doc.SetScalar(goalKey, string(p.Goal)); err != nil {
		return err
	}
	if err := doc.SetScalar(countsKey, string(p.Counts)); err != nil {
		return err
	}
	if p.By.IsZero() {
		if err := doc.SetScalar(byDateKey, ""); err != nil {
			return err
		}
	} else if err := doc.SetDay(byDateKey, p.By); err != nil {
		return err
	}
	for _, one := range []struct {
		key   string
		value any
	}{
		{minutesADayKey, p.MinutesADay},
		{newADayKey, p.NewADay},
		{reviewsADayKey, p.ReviewsADay},
		{backlogKey, p.Backlog},
		{retentionKey, p.Retention},
		{evenLoadKey, p.EvenLoad},
	} {
		if err := doc.SetValue(one.key, one.value); err != nil {
			return err
		}
	}
	shares := make([]markdown.Entry, 0, len(p.Load))
	for _, weekday := range week {
		if share, named := p.Load[weekday]; named {
			shares = append(shares, markdown.Entry{Key: history.DayName(weekday), Value: share})
		}
	}
	return doc.SetMapping(loadKey, shares)
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
