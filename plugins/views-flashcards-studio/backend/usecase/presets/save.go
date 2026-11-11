package presets

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/markdown"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/note"
)

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
		{MinutesADayKey, p.MinutesADay, was.MinutesADay},
		{newADayKey, p.NewADay, was.NewADay},
		{reviewsADayKey, p.ReviewsADay, was.ReviewsADay},
		{backlogKey, p.Backlog, was.Backlog},
		{RetentionKey, p.Retention, was.Retention},
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
		{MinutesADayKey, float64(p.MinutesADay), review.GetMinutesADayBounds()},
		{newADayKey, float64(p.NewADay), review.GetNewADayBounds()},
		{reviewsADayKey, float64(p.ReviewsADay), review.GetReviewsADayBounds()},
		{RetentionKey, p.Retention, review.GetRetentionBounds()},
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
	if ports.IsNotExist(err) {
		return note.ErrNoNote
	}
	return err
}
