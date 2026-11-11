package presets

import (
	"context"
	"errors"
	"io/fs"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/markdown"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/note"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

func TestPresetsInternalsAndErrors(t *testing.T) {
	ctx := t.Context()
	v := domain.Vault{ID: "v1"}
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	day := review.Day{Starts: 4 * time.Hour, In: time.UTC}
	errBoom := errors.New("boom")

	p0 := New(nil, nil, nil, nil, nil, clock.NewFake(now))
	if p0.day.Starts != 0 || p0.day.In != nil {
		t.Fatalf("expected zero day on New, got %+v", p0.day)
	}
	pDay := p0.WithDay(day)
	if pDay.day.Starts != 4*time.Hour {
		t.Fatalf("expected WithDay to set day, got %+v", pDay.day)
	}

	def := CreateDefault()
	if def.FileStat.Size != 0 {
		t.Fatalf("expected zero FileStat.Size on CreateDefault, got %d", def.FileStat.Size)
	}

	// Test getStopReasons returning both overall and today's stop reason.
	pausedPreset := review.Defaults()
	pausedPreset.Goal = review.GoalDate
	pausedPreset.By = time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	overallStop, todayStop := pDay.getStopReasons(pausedPreset)
	if overallStop != review.StoppedPastDay || todayStop != review.StoppedPastDay {
		t.Fatalf("expected StoppedPastDay for both stop reasons, got %q, %q", overallStop, todayStop)
	}

	// Test List: GetPathsOfType error, empty paths, GetNotes error.
	if _, err := New(nil, nil, nil, stubNoteQueries{
		pathsErr: map[domain.NoteType]error{format.TypePreset: errBoom},
	}, nil, nil).List(ctx, v); !errors.Is(err, errBoom) {
		t.Fatalf("expected GetPathsOfType error in List, got %v", err)
	}
	emptyList, err := New(nil, nil, nil, stubNoteQueries{
		paths: map[domain.NoteType][]string{format.TypePreset: nil},
	}, nil, nil).List(ctx, v)
	if err != nil || emptyList != nil {
		t.Fatalf("expected nil, nil for empty preset paths, got %v, %v", emptyList, err)
	}
	if _, err := New(nil, nil, nil, stubNoteQueries{
		paths:    map[domain.NoteType][]string{format.TypePreset: {"P.md"}},
		notesErr: errBoom,
	}, nil, nil).List(ctx, v); !errors.Is(err, errBoom) {
		t.Fatalf("expected GetNotes error in List, got %v", err)
	}

	// Test GetForDeck error branches: links error, roleless problems error, readFirstLink error, multiple links delta.
	if _, err := New(nil, nil, stubLinkQueries{linksErr: errBoom}, nil, nil, clock.NewFake(now)).GetForDeck(ctx, v, "decks/A.md"); !errors.Is(err, errBoom) {
		t.Fatalf("expected links error in GetForDeck, got %v", err)
	}
	pProbErr := New(nil, nil, stubLinkQueries{}, nil, nil, clock.NewFake(now))
	pProbErr.Problems = stubProblemQueries{err: errBoom}
	if _, err := pProbErr.GetForDeck(ctx, v, "decks/A.md"); !errors.Is(err, errBoom) {
		t.Fatalf("expected problems error in GetForDeck, got %v", err)
	}
	twoDangling := stubLinkQueries{
		links: map[string][]domain.ResolvedLink{
			"decks/A.md": {
				{Link: domain.Link{Type: LinkType, Target: domain.Address{Scheme: domain.SchemeName, Value: "Missing1"}}, To: ""},
				{Link: domain.Link{Type: LinkType, Target: domain.Address{Scheme: domain.SchemeName, Value: "Missing2"}}, To: ""},
			},
		},
	}
	gotDangling, err := New(nil, nil, twoDangling, nil, nil, clock.NewFake(now)).GetForDeck(ctx, v, "decks/A.md")
	if err != nil || len(gotDangling.Problems) != 2 || !strings.Contains(gotDangling.Problems[1], "scheduled by Missing1") {
		t.Fatalf("expected two problems naming Missing1 as the scheduling preset, got %+v, %v", gotDangling.Problems, err)
	}

	twoResolved := stubLinkQueries{
		links: map[string][]domain.ResolvedLink{
			"decks/A.md": {
				{Link: domain.Link{Type: LinkType, Target: domain.Address{Scheme: domain.SchemeName, Value: "P"}}, To: "presets/P.md"},
				{Link: domain.Link{Type: LinkType, Target: domain.Address{Scheme: domain.SchemeName, Value: "Q"}}, To: "presets/Q.md"},
			},
		},
	}
	if _, err := New(stubVaultReaders{openErr: errBoom}, nil, twoResolved, nil, nil, clock.NewFake(now)).GetForDeck(ctx, v, "decks/A.md"); !errors.Is(err, errBoom) {
		t.Fatalf("expected read error in readFirstLink, got %v", err)
	}
	readerPreset := stubVaultReader{
		statFn: func(_ context.Context, path string) (domain.FileStat, error) {
			return domain.FileStat{Path: path, Kind: domain.KindNote, Size: 20}, nil
		},
		readFn: func(_ context.Context, _ string) ([]byte, error) {
			return []byte("---\ntype: preset\n---\n"), nil
		},
	}
	gotTwoResolved, err := New(stubVaultReaders{reader: readerPreset}, nil, twoResolved, nil, nil, clock.NewFake(now)).GetForDeck(ctx, v, "decks/A.md")
	if err != nil || len(gotTwoResolved.Problems) != 1 || !strings.Contains(gotTwoResolved.Problems[0], "scheduled by presets/P.md") {
		t.Fatalf("expected delta presets/P.md in multiple-presets problem, got %+v, %v", gotTwoResolved.Problems, err)
	}

	// Test Read / readPreset / parsePresetBytes branches:
	// missing note (Size==0), Open error, Kind != KindNote, Size == MaxBytes vs MaxBytes+1,
	// StatFile ErrNotExist but Read succeeds (!target && err == nil -> NotANote),
	// invalid UTF-8 -> NotText, unreadable frontmatter -> Unreadable.
	if _, err := New(stubVaultReaders{openErr: errBoom}, nil, nil, nil, nil, clock.NewFake(now)).Read(ctx, v, "P.md"); !errors.Is(err, errBoom) {
		t.Fatalf("expected Open error in Read, got %v", err)
	}
	readerMissing := stubVaultReader{
		statFn: func(_ context.Context, _ string) (domain.FileStat, error) { return domain.FileStat{}, fs.ErrNotExist },
		readFn: func(_ context.Context, _ string) ([]byte, error) { return nil, fs.ErrNotExist },
	}
	gotMissing, err := New(stubVaultReaders{reader: readerMissing}, nil, nil, nil, nil, clock.NewFake(now)).Read(ctx, v, "P.md")
	if err != nil || gotMissing.Outcome != note.Missing || gotMissing.FileStat.Size != 0 {
		t.Fatalf("expected Missing with Size=0, got %+v, %v", gotMissing, err)
	}
	readerKindNotNote := stubVaultReader{
		statFn: func(_ context.Context, path string) (domain.FileStat, error) {
			return domain.FileStat{Path: path, Kind: "binary", Size: 10}, nil
		},
		readFn: func(_ context.Context, _ string) ([]byte, error) { return nil, nil },
	}
	gotNotNote, err := New(stubVaultReaders{reader: readerKindNotNote}, nil, nil, nil, nil, clock.NewFake(now)).Read(ctx, v, "P.bin")
	if err != nil || gotNotNote.Outcome != note.NotANote || gotNotNote.FileStat.Path != "P.bin" {
		t.Fatalf("expected NotANote with FileStat set, got %+v, %v", gotNotNote, err)
	}
	readerTooLarge := stubVaultReader{
		statFn: func(_ context.Context, path string) (domain.FileStat, error) {
			return domain.FileStat{Path: path, Kind: domain.KindNote, Size: note.MaxBytes + 1}, nil
		},
		readFn: func(_ context.Context, _ string) ([]byte, error) { return nil, nil },
	}
	gotTooLarge, err := New(stubVaultReaders{reader: readerTooLarge}, nil, nil, nil, nil, clock.NewFake(now)).Read(ctx, v, "P.md")
	if err != nil || gotTooLarge.Outcome != note.TooLarge {
		t.Fatalf("expected TooLarge for MaxBytes+1, got %+v, %v", gotTooLarge, err)
	}
	readerExactMax := stubVaultReader{
		statFn: func(_ context.Context, path string) (domain.FileStat, error) {
			return domain.FileStat{Path: path, Kind: domain.KindNote, Size: note.MaxBytes}, nil
		},
		readFn: func(_ context.Context, _ string) ([]byte, error) { return []byte("---\ntype: preset\n---\n"), nil },
	}
	gotExactMax, err := New(stubVaultReaders{reader: readerExactMax}, nil, nil, nil, nil, clock.NewFake(now)).Read(ctx, v, "P.md")
	if err != nil || gotExactMax.Outcome != note.Ok {
		t.Fatalf("expected Ok for exact MaxBytes, got %+v, %v", gotExactMax, err)
	}
	readerGhost := stubVaultReader{
		statFn: func(_ context.Context, _ string) (domain.FileStat, error) { return domain.FileStat{}, fs.ErrNotExist },
		readFn: func(_ context.Context, _ string) ([]byte, error) { return []byte("---\ntype: preset\n---\n"), nil },
	}
	gotGhost, err := New(stubVaultReaders{reader: readerGhost}, nil, nil, nil, nil, clock.NewFake(now)).Read(ctx, v, "P.md")
	if err != nil || gotGhost.Outcome != note.NotANote {
		t.Fatalf("expected NotANote when StatFile says not exist but Read succeeds, got %+v, %v", gotGhost, err)
	}
	gotNotText, err := parsePresetBytes(Contents{Path: "P.md"}, domain.FileStat{}, []byte{0xff, 0xfe})
	if err != nil || gotNotText.Outcome != note.NotText {
		t.Fatalf("expected NotText for invalid UTF-8, got %+v, %v", gotNotText, err)
	}
	gotUnreadable, err := parsePresetBytes(Contents{Path: "P.md"}, domain.FileStat{}, []byte("---\n: : bad yaml\n---\n"))
	if err != nil || gotUnreadable.Outcome != note.Unreadable {
		t.Fatalf("expected Unreadable for broken frontmatter, got %+v, %v", gotUnreadable, err)
	}

	// Test getPresetAddress branches: notes == nil, Read error, pos.Outcome != note.Ok.
	if _, err := (Presets{}).getPresetAddress(ctx, v, "P.md"); !errors.Is(err, ErrNoPresets) {
		t.Fatalf("expected ErrNoPresets when notes == nil, got %v", err)
	}
	if _, err := New(stubVaultReaders{openErr: errBoom}, nil, nil, stubNoteQueries{}, nil, clock.NewFake(now)).getPresetAddress(ctx, v, "P.md"); !errors.Is(err, errBoom) {
		t.Fatalf("expected Read error in getPresetAddress, got %v", err)
	}
	if _, err := New(stubVaultReaders{reader: readerTooLarge}, nil, nil, stubNoteQueries{}, nil, clock.NewFake(now)).getPresetAddress(ctx, v, "P.md"); !errors.Is(err, ErrNotAPreset) {
		t.Fatalf("expected ErrNotAPreset when Outcome is TooLarge, got %v", err)
	}

	// Test Save / save error branches: Lock error, release error, readers.Open error, StatFile error,
	// Size == MaxBytes vs MaxBytes+1, expectedFp.IsZero(), Read error, writers.Open error.
	validPreset := review.Defaults()
	if _, err := New(nil, stubVaultWriters{lockErr: errBoom}, nil, nil, nil, nil).Save(ctx, v, "P.md", validPreset, domain.FileStat{}); !errors.Is(err, errBoom) {
		t.Fatalf("expected lock error in Save, got %v", err)
	}
	if _, err := New(stubVaultReaders{openErr: errBoom}, stubVaultWriters{}, nil, nil, nil, nil).Save(ctx, v, "P.md", validPreset, domain.FileStat{}); !errors.Is(err, errBoom) {
		t.Fatalf("expected readers.Open error in Save, got %v", err)
	}
	if _, err := New(stubVaultReaders{reader: readerMissing}, stubVaultWriters{}, nil, nil, nil, nil).Save(ctx, v, "P.md", validPreset, domain.FileStat{}); !errors.Is(err, note.ErrNoNote) {
		t.Fatalf("expected ErrNoNote from StatFile in Save, got %v", err)
	}
	if _, err := New(stubVaultReaders{reader: readerTooLarge}, stubVaultWriters{}, nil, nil, nil, nil).Save(ctx, v, "P.md", validPreset, domain.FileStat{}); !errors.Is(err, note.ErrTooLarge) {
		t.Fatalf("expected ErrTooLarge in Save, got %v", err)
	}
	readerReadErr := stubVaultReader{
		statFn: func(_ context.Context, path string) (domain.FileStat, error) {
			return domain.FileStat{Path: path, Kind: domain.KindNote, Size: note.MaxBytes}, nil
		},
		readFn: func(_ context.Context, _ string) ([]byte, error) { return nil, fs.ErrNotExist },
	}
	if _, err := New(stubVaultReaders{reader: readerReadErr}, stubVaultWriters{}, nil, nil, nil, nil).Save(ctx, v, "P.md", validPreset, domain.FileStat{}); !errors.Is(err, note.ErrNoNote) {
		t.Fatalf("expected ErrNoNote from Read in Save, got %v", err)
	}
	if _, err := New(stubVaultReaders{reader: readerExactMax}, stubVaultWriters{openErr: errBoom}, nil, nil, nil, nil).Save(ctx, v, "P.md", validPreset, domain.FileStat{}); !errors.Is(err, errBoom) {
		t.Fatalf("expected writers.Open error in Save, got %v", err)
	}
	var gotExpectedFp domain.FileStat
	writerCheckFp := stubVaultWriter{
		writeFn: func(_ context.Context, path string, data []byte, expected domain.FileStat) (domain.FileStat, error) {
			gotExpectedFp = expected
			return domain.FileStat{Path: path, Kind: domain.KindNote, Size: int64(len(data))}, nil
		},
	}
	if _, err := New(
		stubVaultReaders{reader: readerExactMax},
		stubVaultWriters{writer: writerCheckFp, releaseErr: errBoom},
		nil, nil,
		func(_ context.Context, _ domain.Vault, _ []string) error { return nil },
		nil,
	).Save(ctx, v, "P.md", validPreset, domain.FileStat{}); !errors.Is(err, errBoom) {
		t.Fatalf("expected release error joined in Save, got %v", err)
	}
	if gotExpectedFp.Size != note.MaxBytes {
		t.Fatalf("expected zero stat to default to on-disk FileStat (Size=%d), got %+v", note.MaxBytes, gotExpectedFp)
	}

	// Test writeSettings / writeByDate clearing by_date when p.By.IsZero(), and maps.Equal(p.Load, was.Load).
	doc, err := markdown.Open([]byte("---\ntype: preset\ngoal: by_date\nby_date: 2026-07-01\n---\n"))
	if err != nil {
		t.Fatal(err)
	}
	wasWithDate := review.Defaults()
	wasWithDate.Goal = review.GoalDate
	wasWithDate.By = time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	nowNoDate := review.Defaults()
	if err := writeSettings(doc, nowNoDate, wasWithDate); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(doc.Bytes()), "2026-07-01") {
		t.Fatalf("expected by_date cleared when p.By is zero, got %q", string(doc.Bytes()))
	}

	// Test getDayEntries with non-day keys, unparsed day keys, duplicate day spellings ("Mon" vs "mon"), and full day names that are no longer day names.
	entries := []string{"not-a-day", "tuesday", "tue", "Mon", "mon", "wed", "wednesday"}
	readLoad := map[time.Weekday]int{time.Monday: 100, time.Wednesday: 50}
	dayEntries := getDayEntries(entries, readLoad)
	if dayEntries[time.Monday] != "mon" || dayEntries[time.Wednesday] != "wed" || len(dayEntries) != 2 {
		t.Fatalf("unexpected getDayEntries: %+v", dayEntries)
	}

	// Test checkBounds / checkLoad with invalid Counts and out-of-range time.Weekday keys (< Sunday, > Saturday).
	badCounts := review.Defaults()
	badCounts.Counts = "invalid"
	if err := checkBounds(badCounts); !errors.Is(err, ErrOutOfBounds) {
		t.Fatalf("expected ErrOutOfBounds for invalid Counts, got %v", err)
	}
	if err := checkLoad(map[time.Weekday]int{time.Weekday(-1): 100}); !errors.Is(err, ErrOutOfBounds) {
		t.Fatalf("expected ErrOutOfBounds for weekday < Sunday, got %v", err)
	}
	if err := checkLoad(map[time.Weekday]int{time.Weekday(7): 100}); !errors.Is(err, ErrOutOfBounds) {
		t.Fatalf("expected ErrOutOfBounds for weekday > Saturday, got %v", err)
	}
}
