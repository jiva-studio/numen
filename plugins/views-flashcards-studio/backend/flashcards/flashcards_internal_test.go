package flashcards

import (
	"container/list"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/markdown"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/vaulttest"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/note"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

type stubVaultReaders struct {
	openErr error
	reader  ports.VaultReader
}

func (s stubVaultReaders) Open(_ domain.Vault) (ports.VaultReader, error) {
	if s.openErr != nil {
		return nil, s.openErr
	}
	return s.reader, nil
}

type stubVaultReader struct {
	statFn func(ctx context.Context, path string) (domain.FileStat, error)
	readFn func(ctx context.Context, path string) ([]byte, error)
}

func (s stubVaultReader) StatFile(ctx context.Context, path string) (domain.FileStat, error) {
	return s.statFn(ctx, path)
}

func (s stubVaultReader) Read(ctx context.Context, path string) ([]byte, error) {
	return s.readFn(ctx, path)
}

type stubVaultWriters struct {
	openErr    error
	lockErr    error
	releaseErr error
	writer     ports.VaultWriter
}

func (s stubVaultWriters) Open(_ domain.Vault) (ports.VaultWriter, error) {
	if s.openErr != nil {
		return nil, s.openErr
	}
	return s.writer, nil
}

func (s stubVaultWriters) Lock(_ context.Context, _ domain.Vault) (func() error, error) {
	if s.lockErr != nil {
		return nil, s.lockErr
	}
	return func() error { return s.releaseErr }, nil
}

type stubVaultWriter struct {
	writeFn func(ctx context.Context, path string, data []byte, expected domain.FileStat) (domain.FileStat, error)
}

func (s stubVaultWriter) Create(context.Context, string, []byte) error { return nil }

func (s stubVaultWriter) Write(
	ctx context.Context, path string, data []byte, expected domain.FileStat,
) (domain.FileStat, error) {
	if s.writeFn != nil {
		return s.writeFn(ctx, path, data, expected)
	}
	return domain.FileStat{Path: path, Kind: domain.KindNote, Size: int64(len(data))}, nil
}

type stubLinkQueries struct {
	linksErr     error
	backlinksErr error
	links        map[string][]domain.ResolvedLink
	backlinks    map[string][]domain.ResolvedLink
}

func (s stubLinkQueries) GetLinks(_ context.Context, _ domain.VaultID, path string) ([]domain.ResolvedLink, error) {
	if s.linksErr != nil {
		return nil, s.linksErr
	}
	return s.links[path], nil
}

func (s stubLinkQueries) ResolveLinks(context.Context, domain.VaultID, string, []string) (map[string]domain.ResolvedLink, error) {
	return nil, nil
}

func (s stubLinkQueries) GetBacklinks(_ context.Context, _ domain.VaultID, path string) ([]domain.ResolvedLink, error) {
	if s.backlinksErr != nil {
		return nil, s.backlinksErr
	}
	return s.backlinks[path], nil
}

type stubNoteQueries struct {
	hasErr   error
	has      bool
	pathsErr map[domain.NoteType]error
	paths    map[domain.NoteType][]string
	notesErr error
	notes    map[string]domain.NoteRef
	typesErr error
	types    map[string]domain.NoteType
	namesErr error
	names    map[string][]string
}

func (s stubNoteQueries) Has(_ context.Context, _ domain.VaultID) (bool, error) {
	return s.has, s.hasErr
}

func (s stubNoteQueries) GetPathsOfType(
	_ context.Context, _ domain.VaultID, kind domain.NoteType,
) ([]string, error) {
	if err := s.pathsErr[kind]; err != nil {
		return nil, err
	}
	return s.paths[kind], nil
}

func (s stubNoteQueries) GetNotes(
	_ context.Context, _ domain.VaultID, _ []string,
) (map[string]domain.NoteRef, error) {
	if s.notesErr != nil {
		return nil, s.notesErr
	}
	return s.notes, nil
}

func (s stubNoteQueries) GetTypes(
	_ context.Context, _ domain.VaultID, _ []string,
) (map[string]domain.NoteType, error) {
	if s.typesErr != nil {
		return nil, s.typesErr
	}
	return s.types, nil
}

func (s stubNoteQueries) GetNamedPaths(
	_ context.Context, _ domain.VaultID, name string,
) ([]string, error) {
	if s.namesErr != nil {
		return nil, s.namesErr
	}
	return s.names[name], nil
}

func (s stubNoteQueries) GetCardModels(context.Context, domain.VaultID) ([]ports.CardModel, error) {
	return nil, nil
}

type stubProblemQueries struct {
	err      error
	problems []domain.VaultProblem
}

func (s stubProblemQueries) GetParseProblems(_ context.Context, _ domain.VaultID) ([]domain.VaultProblem, error) {
	return s.problems, s.err
}

type stubLogStores struct {
	openErr error
	store   ports.LogStore
}

func (s stubLogStores) Open(_ domain.Vault) (ports.LogStore, error) {
	if s.openErr != nil {
		return nil, s.openErr
	}
	return s.store, nil
}

type stubLogStore struct {
	listErr   error
	entries   []ports.Entry
	readFn    func(ctx context.Context, run string) ([]byte, error)
	appendErr error
}

func (s stubLogStore) ListLogs(_ context.Context) ([]ports.Entry, error) {
	return s.entries, s.listErr
}

func (s stubLogStore) ReadLog(ctx context.Context, run string) ([]byte, error) {
	if s.readFn != nil {
		return s.readFn(ctx, run)
	}
	return nil, nil
}

func (s stubLogStore) AppendLog(_ context.Context, _ string, _ []byte) error {
	return s.appendErr
}

type stubScheduleStore struct {
	readErr  error
	writeErr error
	data     map[domain.VaultID][]byte
}

func (s *stubScheduleStore) Read(_ context.Context, id domain.VaultID) ([]byte, error) {
	if s.readErr != nil {
		return nil, s.readErr
	}
	raw, ok := s.data[id]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return raw, nil
}

func (s *stubScheduleStore) Write(_ context.Context, id domain.VaultID, data []byte) error {
	if s.writeErr != nil {
		return s.writeErr
	}
	if s.data == nil {
		s.data = make(map[domain.VaultID][]byte)
	}
	s.data[id] = slices.Clone(data)
	return nil
}

type stubAlgorithm struct{}

func (stubAlgorithm) GetName() string { return "stub" }

func (stubAlgorithm) CreateScheduler(retention float64) review.Scheduler {
	return review.NewFSRSAt(retention)
}

func TestBudgetInternals(t *testing.T) {
	ctx := t.Context()
	v := domain.Vault{ID: "v1"}
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	day := review.Day{Starts: 0, In: time.UTC}

	errLinks := errors.New("links failed")
	readsErr := Presets{links: stubLinkQueries{linksErr: errLinks}}.OpenReads()
	c1 := ReviewCard{ID: review.CardID{Card: "c1", Template: "T"}, Deck: "decks/A.md"}
	c2 := ReviewCard{ID: review.CardID{Card: "c2", Template: "T"}, Deck: "decks/B.md"}
	if _, err := getBudgets(ctx, v, budgetInputs{
		reads:       readsErr,
		day:         day,
		reviewCards: []ReviewCard{c1},
		schedules:   map[review.CardID]review.Schedule{},
		delta:       review.NewFSRS(),
		pos:         newSchedulerAt,
		now:         now,
	}); !errors.Is(err, errLinks) {
		t.Fatalf("expected links error from getBudgets, got %v", err)
	}

	env := vaulttest.New(t, map[string]string{
		"DatePreset.md": "---\ntype: preset\ngoal: by_date\nby_date: 2026-07-10\nnew_a_day: 100\nreviews_a_day: 100\n---\n",
		"decks/A.md":    "---\ntype: deck\nlinks:\n  - to: DatePreset\n    role: ref\n    type: preset\n---\n",
		"decks/B.md":    "---\ntype: deck\nlinks:\n  - to: DatePreset\n    role: ref\n    type: preset\n---\n",
	})
	presets := NewPresets(env.Ports.Readers, env.Ports.Writers, env.Queries, env.Queries, scanOf(env), clock.NewFake(now)).WithDay(day)
	b, err := getBudgets(ctx, env.Vault, budgetInputs{
		reads:       presets.OpenReads(),
		day:         day,
		reviewCards: []ReviewCard{c1, c2},
		schedules:   map[review.CardID]review.Schedule{},
		delta:       review.NewFSRS(),
		pos:         newSchedulerAt,
		now:         now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if b.decks["decks/A.md"] != "DatePreset.md" {
		t.Fatalf("expected decks map populated, got %+v", b.decks)
	}
	if b.allowances["DatePreset.md"].admits.Keeps.New != 1 {
		t.Fatalf("expected date goal with time to learn to pace 2 cards at 1/day, got %d",
			b.allowances["DatePreset.md"].admits.Keeps.New)
	}

	envNear := vaulttest.New(t, map[string]string{
		"DatePreset.md": "---\ntype: preset\ngoal: by_date\nby_date: 2026-06-12\nnew_a_day: 100\nreviews_a_day: 100\n---\n",
		"decks/A.md":    "---\ntype: deck\nlinks:\n  - to: DatePreset\n    role: ref\n    type: preset\n---\n",
		"decks/B.md":    "---\ntype: deck\nlinks:\n  - to: DatePreset\n    role: ref\n    type: preset\n---\n",
	})
	presetsNear := NewPresets(envNear.Ports.Readers, envNear.Ports.Writers, envNear.Queries, envNear.Queries, scanOf(envNear), clock.NewFake(now)).WithDay(day)
	bNear, err := getBudgets(ctx, envNear.Vault, budgetInputs{
		reads:       presetsNear.OpenReads(),
		day:         day,
		reviewCards: []ReviewCard{c1, c2},
		schedules:   map[review.CardID]review.Schedule{},
		delta:       review.NewFSRS(),
		pos:         newSchedulerAt,
		now:         now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if bNear.allowances["DatePreset.md"].admits.Keeps.New != 2 {
		t.Fatalf("expected date goal with no time to learn to begin both unbegun cards today, got %d",
			bNear.allowances["DatePreset.md"].admits.Keeps.New)
	}

	totals := sumSpentByPreset(map[string]review.Spent{
		"decks/A.md": {Answered: 2, New: 1, Reviews: 3, Took: 5 * time.Second},
		"decks/B.md": {Answered: 4, New: 2, Reviews: 6, Took: 7 * time.Second},
	}, map[string]string{
		"decks/A.md": "P.md",
		"decks/B.md": "P.md",
	})
	if totals["P.md"].Reviews != 9 || totals["P.md"].New != 3 || totals["P.md"].Answered != 6 || totals["P.md"].Took != 12*time.Second {
		t.Fatalf("unexpected sumSpentByPreset: %+v", totals["P.md"])
	}

	share := &allowance{
		admits: review.Allowance{New: 5, Reviews: 5, Minutes: time.Minute},
		cost:   review.AnswerCost{New: time.Second, Review: time.Second},
		counts: review.BudgetUnitCards,
	}
	bCharge := &budgets{
		reviewCardPresets: map[review.CardID]string{c1.ID: "missing.md", c2.ID: "paused.md"},
		allowances: map[string]*allowance{
			"paused.md": {
				admits: review.Allowance{New: 5, Reviews: 5, Minutes: time.Minute, Stops: review.StoppedNoMinutes},
			},
		},
	}
	if bCharge.charge(share, c1.ID, false) {
		t.Fatal("expected false charging card with missing preset")
	}
	if bCharge.charge(share, c2.ID, false) {
		t.Fatal("expected false charging card with paused preset")
	}

	bStop := &budgets{
		allowances: map[string]*allowance{
			"none.md":   {spentStats: review.Spent{Answered: 0}},
			"spent1.md": {spentStats: review.Spent{Answered: 1}},
		},
	}
	if err := bStop.getStopError("none.md"); err == nil || !strings.Contains(err.Error(), "no cards due") {
		t.Fatalf("expected no cards due error for Answered=0, got %v", err)
	}
	if err := bStop.getStopError("spent1.md"); err == nil || !strings.Contains(err.Error(), "daily budget spent") {
		t.Fatalf("expected daily budget spent error for Answered=1, got %v", err)
	}

	bAsk := &budgets{
		reviewCardPresets: map[review.CardID]string{c1.ID: "P.md", c2.ID: "P.md"},
		allowances: map[string]*allowance{
			"P.md": {
				admits: review.Allowance{
					Keeps:   review.Budget{New: 10, Reviews: 10, Minutes: 10},
					New:     10,
					Reviews: 10,
					Minutes: 10 * time.Minute,
				},
				cost:   review.AnswerCost{New: time.Second, Review: time.Second},
				counts: review.BudgetUnitCards,
			},
		},
	}
	candidates := bAsk.getAsking(
		[]ReviewCard{c1, c2},
		map[review.CardID]review.Schedule{
			c1.ID: {Due: now.Add(-time.Hour), Reps: 1},
			c2.ID: {Due: now.Add(-time.Hour), Reps: 1},
		},
		day, now, Scope{Deck: "decks/A.md"},
	)
	if len(candidates.dueCards) != 1 || candidates.dueCards[0].Deck != "decks/A.md" {
		t.Fatalf("expected only decks/A.md due card in scope, got %+v", candidates.dueCards)
	}

	ds := &deckShare{deck: "decks/A.md", dueCards: []int{0, 1}, newCards: []int{0, 1}, debt: 2, begun: 3}
	rem := ds.countRemaining(budgetAllocation{dueCards: []bool{true, false}, newCards: []bool{false, true}})
	if rem != 2 {
		t.Fatalf("expected 2 remaining cards, got %d", rem)
	}
	bTime := &budgets{
		spentByDeck: map[string]review.Spent{"decks/A.md": {Took: 10 * time.Second}},
	}
	gotTime := bTime.getTimeSpent(ds, review.AnswerCost{Review: 3 * time.Second, New: 4 * time.Second})
	if gotTime != 28*time.Second {
		t.Fatalf("expected 28s time spent, got %v", gotTime)
	}

	reg := &shareRegistry{shares: make(map[string]map[string]*deckShare)}
	gotShare := reg.getShare("P.md", "decks/A.md")
	if !reflect.DeepEqual(*gotShare, deckShare{deck: "decks/A.md"}) {
		t.Fatalf("expected zero-initialized deckShare, got %+v", *gotShare)
	}

	bSpend := &budgets{
		reviewCardPresets: map[review.CardID]string{c1.ID: "01-paused.md", c2.ID: "02-active.md"},
		decks:             map[string]string{"decks/C.md": "02-active.md"},
		spentByDeck:       map[string]review.Spent{"decks/C.md": {Reviews: 1, New: 1, Took: time.Second}},
		allowances: map[string]*allowance{
			"01-paused.md": {
				admits: review.Allowance{Stops: review.StoppedNoMinutes},
			},
			"02-active.md": {
				admits: review.Allowance{
					Keeps:   review.Budget{New: 5, Reviews: 5, Minutes: 5},
					New:     5,
					Reviews: 5,
					Minutes: 5 * time.Minute,
				},
				cost:   review.AnswerCost{New: time.Second, Review: time.Second},
				counts: review.BudgetUnitCards,
			},
		},
	}
	collected := bSpend.collectShares([]ReviewCard{c2}, []ReviewCard{c1})
	if !slices.Equal(collected.order, []string{"01-paused.md", "02-active.md"}) {
		t.Fatalf("expected sorted shares order, got %v", collected.order)
	}
	if _, hasC := collected.shares["02-active.md"]["decks/C.md"]; !hasC {
		t.Fatal("expected deck with prior activity in shares")
	}
	alloc := bSpend.spend([]ReviewCard{c2}, []ReviewCard{c1})
	if !alloc.dueCards[0] || alloc.newCards[0] {
		t.Fatalf("expected active preset card allocated and paused skipped, got %+v", alloc)
	}

	emptyShare := &deckShare{deck: "decks/C.md", dueIndex: 5, newIndex: 5}
	emptyAlloc := budgetAllocation{}
	bSpend.spendPreset(bSpend.allowances["02-active.md"], []*deckShare{emptyShare}, nil, nil, &emptyAlloc)
	if emptyShare.dueIndex != 0 || emptyShare.newIndex != 0 {
		t.Fatalf("expected spendPreset to reset dueIndex and newIndex to 0, got %d, %d",
			emptyShare.dueIndex, emptyShare.newIndex)
	}

	bDiv := &budgets{
		spentByDeck: map[string]review.Spent{
			"decks/A.md": {Reviews: 2, New: 1, Took: 10 * time.Second},
			"decks/B.md": {Reviews: 5, New: 5, Took: 100 * time.Second},
		},
	}
	oneAllow := &allowance{
		admits: review.Allowance{
			Keeps: review.Budget{Reviews: 4, New: 2, Minutes: 1},
		},
		cost: review.AnswerCost{Review: 5 * time.Second, New: 6 * time.Second},
	}
	divShares := bDiv.divide(oneAllow, []*deckShare{
		{deck: "decks/A.md", dueCards: []int{0, 1}, newCards: []int{0}},
		{deck: "decks/B.md", dueCards: nil, newCards: nil},
	})
	if divShares[0].admits.Reviews != 0 || divShares[1].admits.Reviews != 0 {
		t.Fatalf("expected clamped 0 review shares, got %d and %d",
			divShares[0].admits.Reviews, divShares[1].admits.Reviews)
	}
	if divShares[1].admits.New != 0 || divShares[1].admits.Minutes != 0 {
		t.Fatalf("expected clamped 0 new and minutes on deck B, got %+v", divShares[1].admits)
	}

	advShare := &deckShare{dueCards: []int{0, 1}, newCards: []int{0}}
	advShare.advancePastOthers(budgetAllocation{dueCards: []bool{true, false}, newCards: []bool{true}})
	if advShare.dueIndex != 1 || advShare.newIndex != 1 {
		t.Fatalf("expected advancePastOthers to advance to 1,1, got %d,%d", advShare.dueIndex, advShare.newIndex)
	}
}

func TestCardsDueInternalsAndErrors(t *testing.T) {
	ctx := t.Context()
	v := domain.Vault{ID: "v1"}
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	day := review.Day{Starts: 0, In: time.UTC}
	errBoom := errors.New("boom")

	c1 := ReviewCard{ID: review.CardID{Card: "c1", Template: "T"}, Deck: "decks/B.md"}
	c2 := ReviewCard{ID: review.CardID{Card: "c2", Template: "T"}, Deck: "decks/A.md"}
	rows := make(deckRows)
	bDue := &budgets{
		reviewCardPresets: map[review.CardID]string{c1.ID: "P.md", c2.ID: "P.md"},
		cards:             map[string]int{"P.md": 2},
		allowances:        map[string]*allowance{"P.md": {}},
	}
	var out CardsDue
	dueMap, newMap := rows.countDueAndNew(bDue, asking{dueCards: []ReviewCard{c1, c2}}, &out)
	if rows.get("decks/B.md").Due != 1 || rows.get("decks/A.md").Due != 1 {
		t.Fatalf("expected per-deck Due incremented, got %+v, %+v", rows["decks/A.md"], rows["decks/B.md"])
	}
	sortedDecks := rows.getSorted()
	if len(sortedDecks) != 2 || sortedDecks[0].Deck != "decks/A.md" || sortedDecks[1].Deck != "decks/B.md" {
		t.Fatalf("expected sorted decks, got %+v", sortedDecks)
	}
	presetDue := bDue.getCardsDue(dueMap, newMap)
	if len(presetDue) != 1 || presetDue[0].Decks != 0 || presetDue[0].Due != 2 {
		t.Fatalf("unexpected getCardsDue: %+v", presetDue)
	}

	uNoNotes := NewCountCardsDue(ListReviewCards{}, Schedules{}, Presets{}, day, clock.NewFake(now))
	gotNoNotes, err := uNoNotes.getPresets(ctx, v, nil, bDue, dueMap, newMap)
	if err != nil || len(gotNoNotes) != 1 || gotNoNotes[0].Decks != 0 {
		t.Fatalf("expected getPresets without notes to return out directly, got %+v, %v", gotNoNotes, err)
	}

	uDeckErr := NewCountCardsDue(ListReviewCards{
		notes: stubNoteQueries{pathsErr: map[domain.NoteType]error{format.TypeDeck: errBoom}},
	}, Schedules{}, Presets{}, day, clock.NewFake(now))
	if _, err := uDeckErr.getPresets(ctx, v, nil, bDue, dueMap, newMap); !errors.Is(err, errBoom) {
		t.Fatalf("expected deck paths error from getPresets, got %v", err)
	}

	uForDeckErr := NewCountCardsDue(ListReviewCards{
		notes: stubNoteQueries{paths: map[domain.NoteType][]string{format.TypeDeck: {"decks/A.md"}}},
	}, Schedules{}, Presets{}, day, clock.NewFake(now))
	readsLinkErr := Presets{links: stubLinkQueries{linksErr: errBoom}}.OpenReads()
	if _, err := uForDeckErr.getPresets(ctx, v, readsLinkErr, bDue, dueMap, newMap); !errors.Is(err, errBoom) {
		t.Fatalf("expected GetForDeck error from getPresets, got %v", err)
	}

	uPresetPathsErr := NewCountCardsDue(ListReviewCards{
		notes: stubNoteQueries{
			paths:    map[domain.NoteType][]string{format.TypeDeck: nil},
			pathsErr: map[domain.NoteType]error{format.TypePreset: errBoom},
		},
	}, Schedules{}, Presets{}, day, clock.NewFake(now))
	if _, err := uPresetPathsErr.getPresets(ctx, v, Presets{}.OpenReads(), bDue, dueMap, newMap); !errors.Is(err, errBoom) {
		t.Fatalf("expected preset paths error from getPresets, got %v", err)
	}

	uPresetReadErr := NewCountCardsDue(ListReviewCards{
		notes: stubNoteQueries{
			paths: map[domain.NoteType][]string{
				format.TypePreset: {"P.md", "Q.md"},
			},
		},
	}, Schedules{}, Presets{}, day, clock.NewFake(now))
	readsOpenErr := Presets{
		readers: stubVaultReaders{openErr: errBoom},
		now:     clock.NewFake(now),
	}.OpenReads()
	if _, err := uPresetReadErr.getPresets(ctx, v, readsOpenErr, bDue, dueMap, newMap); !errors.Is(err, errBoom) {
		t.Fatalf("expected read error for second preset Q.md after pointed P.md, got %v", err)
	}

	env := vaulttest.New(t, map[string]string{
		"Term.md":    "---\ntype: card-model\nfields:\n  - Front\n---\n\n## Card\n\n### Front\n\n{{Front}}\n\n### Back\n\n{{Front}}\n",
		"decks/A.md": "---\ntype: deck\n---\n\n## One ^k7m2xq9fzp\n\n[[Term]]\n\n### Front\n\nhi\n",
	})
	rc := NewListReviewCards(env.Ports.Readers, env.Queries, env.Queries)
	presets := NewPresets(env.Ports.Readers, env.Ports.Writers, env.Queries, env.Queries, scanOf(env), clock.NewFake(now)).WithDay(day)
	sched := NewSchedules(env.Ports.Logs, review.NewFSRS(), day, rc, presets)
	dueUC := NewCountCardsDue(rc, sched, presets, day, clock.NewFake(now))

	if _, err := NewCountCardsDue(
		NewListReviewCards(env.Ports.Readers, stubNoteQueries{hasErr: errBoom}, env.Queries),
		sched, presets, day, clock.NewFake(now),
	).readInputs(ctx, v); !errors.Is(err, errBoom) {
		t.Fatalf("expected reviewCards error in readInputs, got %v", err)
	}

	if _, err := NewCountCardsDue(
		rc,
		NewSchedules(stubLogStores{openErr: errBoom}, review.NewFSRS(), day, rc, presets),
		presets, day, clock.NewFake(now),
	).readInputs(ctx, env.Vault); !errors.Is(err, errBoom) {
		t.Fatalf("expected log read error in readInputs, got %v", err)
	}

	badPresets := NewPresets(env.Ports.Readers, env.Ports.Writers, stubLinkQueries{linksErr: errBoom}, env.Queries, scanOf(env), clock.NewFake(now))
	if _, err := NewCountCardsDue(
		rc,
		NewSchedules(env.Ports.Logs, review.NewFSRS(), day, rc, badPresets),
		badPresets, day, clock.NewFake(now),
	).readInputs(ctx, env.Vault); !errors.Is(err, errBoom) {
		t.Fatalf("expected assignment error in readInputs, got %v", err)
	}

	// Trigger getPresets error at line 124 of Execute by having a preset path fail on read.
	notesWithBadPreset := stubNoteQueries{
		has: true,
		paths: map[domain.NoteType][]string{
			format.TypeDeck:   nil,
			format.TypePreset: {"BadPreset.md"},
		},
	}
	rcEmpty := NewListReviewCards(stubVaultReaders{openErr: errBoom}, notesWithBadPreset, nil)
	presetsOpenFail := NewPresets(stubVaultReaders{openErr: errBoom}, nil, nil, notesWithBadPreset, nil, clock.NewFake(now))
	schedEmpty := NewSchedules(env.Ports.Logs, review.NewFSRS(), day, rcEmpty, presetsOpenFail)
	if _, err := NewCountCardsDue(rcEmpty, schedEmpty, presetsOpenFail, day, clock.NewFake(now)).Execute(ctx, env.Vault); !errors.Is(err, errBoom) {
		t.Fatalf("expected getPresets error in CountCardsDue.Execute, got %v", err)
	}
	_ = dueUC
}

type countingLogStores struct {
	calls      int
	failOnCall int
	err        error
	store      ports.LogStore
}

func (c *countingLogStores) Open(_ domain.Vault) (ports.LogStore, error) {
	c.calls++
	if c.calls == c.failOnCall {
		return nil, c.err
	}
	return c.store, nil
}

type cancelingNoteQueries struct {
	stubNoteQueries
	cancel context.CancelFunc
}

func (c cancelingNoteQueries) GetPathsOfType(
	ctx context.Context, id domain.VaultID, kind domain.NoteType,
) ([]string, error) {
	if c.cancel != nil {
		c.cancel()
	}
	return c.stubNoteQueries.GetPathsOfType(ctx, id, kind)
}

type cancelingScheduleStore struct {
	stubScheduleStore
	cancel context.CancelFunc
}

func (c *cancelingScheduleStore) Read(ctx context.Context, id domain.VaultID) ([]byte, error) {
	if c.cancel != nil {
		c.cancel()
	}
	return c.stubScheduleStore.Read(ctx, id)
}

func TestCardsDueCancellationPaths(t *testing.T) {
	v := domain.Vault{ID: "v1"}
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	day := review.Day{Starts: 0, In: time.UTC}

	ctx1, cancel1 := context.WithCancel(t.Context())
	rcCancel := NewListReviewCards(
		stubVaultReaders{reader: stubVaultReader{}},
		cancelingNoteQueries{
			stubNoteQueries: stubNoteQueries{has: true},
			cancel:          cancel1,
		},
		nil,
	)
	uCancelAfterRC := NewCountCardsDue(rcCancel, Schedules{}, Presets{}, day, clock.NewFake(now))
	if _, err := uCancelAfterRC.readInputs(ctx1, v); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled after reviewCards.Execute, got %v", err)
	}

	ctx2, cancel2 := context.WithCancel(t.Context())
	rcOk := NewListReviewCards(
		stubVaultReaders{reader: stubVaultReader{}},
		stubNoteQueries{has: true},
		nil,
	)
	schedCancel := NewSchedules(
		stubLogStores{store: stubLogStore{}},
		review.NewFSRS(), day, rcOk, Presets{},
	)
	schedCancel.Cache = &cancelingScheduleStore{cancel: cancel2}
	uCancelAfterSched := NewCountCardsDue(rcOk, schedCancel, Presets{}, day, clock.NewFake(now))
	if _, err := uCancelAfterSched.readInputs(ctx2, v); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled after getSchedulesCached, got %v", err)
	}
}

func TestCountReviewsInternalsAndErrors(t *testing.T) {
	ctx := t.Context()
	v := domain.Vault{ID: "v1"}
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	day := review.Day{Starts: 0, In: time.UTC}
	errBoom := errors.New("boom")

	uListFail := NewCountReviews(stubLogStores{openErr: errBoom}, Schedules{}, day, clock.NewFake(now))
	if _, err := uListFail.Execute(ctx, v); !errors.Is(err, errBoom) {
		t.Fatalf("expected GetLogFiles error, got %v", err)
	}

	secondOpenFail := &countingLogStores{failOnCall: 2, err: errBoom, store: stubLogStore{}}
	uOpen2Fail := NewCountReviews(secondOpenFail, Schedules{}, day, clock.NewFake(now))
	if _, err := uOpen2Fail.Execute(ctx, v); !errors.Is(err, errBoom) {
		t.Fatalf("expected second Open error, got %v", err)
	}

	t1 := addTallies(
		review.Tally{Answered: 4, Again: 1, Hard: 1, Good: 1, Easy: 1},
		review.Tally{Answered: 8, Again: 2, Hard: 2, Good: 2, Easy: 2},
	)
	if t1 != (review.Tally{Answered: 12, Again: 3, Hard: 3, Good: 3, Easy: 3}) {
		t.Fatalf("unexpected addTallies: %+v", t1)
	}

	a1 := review.Answer{
		ID:         "01J00000000000000000000001",
		ReviewCard: review.CardID{Card: "c1", Template: "T"},
		At:         now.Add(-48 * time.Hour),
		Rating:     review.Good,
	}
	a2 := review.Answer{
		ID:         "01J00000000000000000000002",
		ReviewCard: review.CardID{Card: "c1", Template: "T"},
		At:         now.Add(-24 * time.Hour),
		Rating:     review.Easy,
	}
	uncounted := getUncounted([]review.Answer{a1, a2}, map[string]bool{a1.ID: true})
	if len(uncounted) != 1 || uncounted[0].ID != a2.ID {
		t.Fatalf("unexpected getUncounted: %+v", uncounted)
	}
	if gotEmpty := getUncounted(nil, map[string]bool{}); gotEmpty == nil || len(gotEmpty) != 0 {
		t.Fatalf("expected non-nil empty slice from getUncounted, got %#v", gotEmpty)
	}

	raw1, err := review.Write(a1)
	if err != nil {
		t.Fatal(err)
	}
	raw2, err := review.Write(a2)
	if err != nil {
		t.Fatal(err)
	}

	// Test countFile with cached run that has repeated IDs and was not opened initially (coming == false),
	// and also test gone/unreadable files not appended to run.log.Answers.
	storeRepeated := stubLogStore{
		entries: []ports.Entry{
			{Run: "01-gone", Size: 10},
			{Run: "02-unreadable", Size: 10},
			{Run: "03-first", Size: len(raw1)},
			{Run: "04-cached-repeat", Size: len(raw1) + len(raw2)},
		},
		readFn: func(_ context.Context, run string) ([]byte, error) {
			switch run {
			case "01-gone":
				return nil, fs.ErrNotExist
			case "02-unreadable":
				return nil, fs.ErrPermission
			case "03-first":
				return raw1, nil
			case "04-cached-repeat":
				return append(slices.Clone(raw1), raw2...), nil
			default:
				return nil, errBoom
			}
		},
	}
	uNoSched := NewCountReviews(stubLogStores{store: storeRepeated}, Schedules{}, day, clock.NewFake(now))
	runState := reviewCounter{
		was: map[string]cachedRun{
			"04-cached-repeat": {
				Name: "04-cached-repeat",
				Size: len(raw1) + len(raw2),
				Days: review.GetDayTallies(day, []review.Answer{a1, a2}),
				IDs:  []string{a1.ID, a2.ID},
			},
		},
		out:        ReviewCounts{Days: make(map[string]review.Tally)},
		countedIDs: make(map[string]bool),
		coming:     false,
	}
	logHelper := Log{Stores: stubLogStores{store: storeRepeated}}
	for _, entry := range storeRepeated.entries {
		if err := uNoSched.countFile(ctx, logHelper, storeRepeated, entry, &runState); err != nil {
			t.Fatal(err)
		}
	}
	if runState.out.Answered != 2 {
		t.Fatalf("expected 2 unique answers counted across repeated cached run, got %d", runState.out.Answered)
	}
	if len(runState.log.Answers) != 1 {
		t.Fatalf("expected only opened valid run appended when coming=false, got %d", len(runState.log.Answers))
	}

	// Test countFile error on lazy read when !didOpen and isRepeated is true.
	storeFailLazy := stubLogStore{
		readFn: func(_ context.Context, _ string) ([]byte, error) {
			return nil, errBoom
		},
	}
	runFailLazy := reviewCounter{
		was: map[string]cachedRun{
			"04-cached-repeat": {
				Name: "04-cached-repeat",
				Size: 10,
				IDs:  []string{a1.ID},
			},
		},
		out:        ReviewCounts{Days: make(map[string]review.Tally)},
		countedIDs: map[string]bool{a1.ID: true},
		coming:     false,
	}
	if err := uNoSched.countFile(
		ctx, Log{Stores: stubLogStores{store: storeFailLazy}}, storeFailLazy,
		ports.Entry{Run: "04-cached-repeat", Size: 10}, &runFailLazy,
	); !errors.Is(err, errBoom) {
		t.Fatalf("expected error on lazy ReadFile for repeated cached run, got %v", err)
	}

	// Test getDueAndRetained when scheduler is nil (returns non-nil falls map),
	// when GetFromLog fails, when schedule IsNew() is skipped, and Retained is returned.
	fallsNil, retNil, err := uNoSched.getDueAndRetained(ctx, v, ReviewLog{})
	if err != nil || fallsNil == nil || retNil != nil {
		t.Fatalf("expected non-nil empty falls and nil retained when scheduler is nil, got %#v, %#v, %v",
			fallsNil, retNil, err)
	}

	schedFail := NewSchedules(
		stubLogStores{store: stubLogStore{}},
		review.NewFSRS(), day,
		NewListReviewCards(stubVaultReaders{}, stubNoteQueries{hasErr: errBoom}, nil),
		Presets{links: stubLinkQueries{}},
	)
	uSchedFail := NewCountReviews(stubLogStores{store: stubLogStore{}}, schedFail, day, clock.NewFake(now))
	if _, _, err := uSchedFail.getDueAndRetained(ctx, v, ReviewLog{}); !errors.Is(err, errBoom) {
		t.Fatalf("expected GetFromLog error from getDueAndRetained, got %v", err)
	}
	if _, err := uSchedFail.Execute(ctx, v); !errors.Is(err, errBoom) {
		t.Fatalf("expected Execute to propagate getDueAndRetained error, got %v", err)
	}

	// Test getDueAndRetained with a cached schedule that is IsNew() (Due is zero) so s.IsNew() is exercised,
	// plus an Easy answer that moves c1 into review and a second answer on it, so Retained is non-empty.
	aEasy := a1
	aEasy.Rating = review.Easy
	rawEasy, err := review.Write(aEasy)
	if err != nil {
		t.Fatal(err)
	}
	schedWithNew := NewSchedules(stubLogStores{store: stubLogStore{
		entries: []ports.Entry{{Run: "01", Size: len(rawEasy) + len(raw2)}},
		readFn: func(_ context.Context, _ string) ([]byte, error) {
			return append(slices.Clone(rawEasy), raw2...), nil
		},
	}}, review.NewFSRS(), day, ListReviewCards{}, Presets{})
	uWithRetained := NewCountReviews(schedWithNew.logs, schedWithNew, day, clock.NewFake(now))
	uWithRetained.Cache = &stubScheduleStore{readErr: errBoom}
	counts, err := uWithRetained.Execute(ctx, v)
	if err != nil {
		t.Fatal(err)
	}
	if len(counts.Retained) == 0 {
		t.Fatalf("expected non-empty Retained map, got %+v", counts.Retained)
	}

	// Test GetReviewDays filtering Due, Retained, and error propagation.
	daysFail := NewGetReviewDays(uListFail)
	if _, err := daysFail.Execute(ctx, v, "", ""); !errors.Is(err, errBoom) {
		t.Fatalf("expected GetReviewDays error propagation, got %v", err)
	}
	daysOK := NewGetReviewDays(uWithRetained).WithDay(day)
	filtered, err := daysOK.Execute(ctx, v, "2099-01-01", "2099-01-02")
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered.Days) != 0 || len(filtered.Due) != 0 || len(filtered.Retained) != 0 {
		t.Fatalf("expected all maps filtered out for 2099 range, got Days=%v Due=%v Retained=%v",
			filtered.Days, filtered.Due, filtered.Retained)
	}
}

func TestCurveInternalsAndErrors(t *testing.T) {
	ctx := t.Context()
	v := domain.Vault{ID: "v1"}
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	day := review.Day{Starts: 0, In: time.UTC}
	errBoom := errors.New("boom")

	pc := NewProjectCurve(ListReviewCards{}, Schedules{}, Presets{}, day, clock.NewFake(now))
	if pc.Cores != 0 || pc.By != nil {
		t.Fatalf("expected Cores=0 and By=nil from NewProjectCurve, got %+v", pc)
	}

	goalDatePreset := review.Defaults()
	goalDatePreset.Goal = review.GoalDate
	goalDatePreset.By = time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	if err := checkGoal(goalDatePreset); err != nil {
		t.Fatalf("expected nil from checkGoal for GoalDate, got %v", err)
	}

	// Test getScheduledDecks branches: path == "", links == nil, backlinks error.
	rcDecks := NewListReviewCards(
		stubVaultReaders{},
		stubNoteQueries{has: true, paths: map[domain.NoteType][]string{format.TypeDeck: {"decks/A.md"}}},
		nil,
	)
	pcNoLinks := NewProjectCurve(rcDecks, Schedules{}, Presets{links: nil}, day, clock.NewFake(now))
	decks1, err := pcNoLinks.getScheduledDecks(ctx, v, "P.md")
	if err != nil || !slices.Equal(decks1, []string{"decks/A.md"}) {
		t.Fatalf("expected all decks when links == nil, got %v, %v", decks1, err)
	}
	pcBacklinkErr := NewProjectCurve(
		rcDecks, Schedules{}, Presets{links: stubLinkQueries{backlinksErr: errBoom}}, day, clock.NewFake(now),
	)
	if _, err := pcBacklinkErr.getScheduledDecks(ctx, v, "P.md"); !errors.Is(err, errBoom) {
		t.Fatalf("expected backlinks error, got %v", err)
	}
	decksDefault, err := pcBacklinkErr.getScheduledDecks(ctx, v, "")
	if err != nil || !slices.Equal(decksDefault, []string{"decks/A.md"}) {
		t.Fatalf("expected all decks when path is empty even if backlinks would fail, got %v, %v", decksDefault, err)
	}

	// Test readScheduledCards error, caching per deck, and continue on non-matching deck.
	readsFail := Presets{links: stubLinkQueries{linksErr: errBoom}}.OpenReads()
	c1 := ReviewCard{ID: review.CardID{Card: "c1", Template: "T"}, Deck: "decks/A.md"}
	c2 := ReviewCard{ID: review.CardID{Card: "c2", Template: "T"}, Deck: "decks/A.md"}
	c3 := ReviewCard{ID: review.CardID{Card: "c3", Template: "T"}, Deck: "decks/B.md"}
	if _, err := pc.readScheduledCards(ctx, v, readsFail, "P.md", []ReviewCard{c1}, nil); !errors.Is(err, errBoom) {
		t.Fatalf("expected readScheduledCards error, got %v", err)
	}
	if _, err := pc.countPointingDecks(ctx, v, readsFail, "P.md", []string{"decks/A.md"}, nil); !errors.Is(err, errBoom) {
		t.Fatalf("expected countPointingDecks error, got %v", err)
	}

	env := vaulttest.New(t, map[string]string{
		"Term.md": "---\ntype: card-model\nfields:\n  - Front\n---\n\n## Card\n\n### Front\n\n{{Front}}\n\n### Back\n\n{{Front}}\n",
		"P.md":    "---\ntype: preset\n---\n",
		"Q.md":    "---\ntype: preset\n---\n",
		"decks/A.md": "---\ntype: deck\nlinks:\n  - to: P\n    role: ref\n    type: preset\n---\n" +
			"\n## One ^k7m2xq9fzp\n\n[[Term]]\n\n### Front\n\na\n\n## Two ^3f4g5h6j7k\n\n[[Term]]\n\n### Front\n\nb\n",
		"decks/B.md": "---\ntype: deck\nlinks:\n  - to: Q\n    role: ref\n    type: preset\n---\n" +
			"\n## Three ^9z8y7x6w5v\n\n[[Term]]\n\n### Front\n\nc\n",
	})
	presets := NewPresets(env.Ports.Readers, env.Ports.Writers, env.Queries, env.Queries, scanOf(env), clock.NewFake(now)).WithDay(day)
	sc, err := pc.readScheduledCards(
		ctx, env.Vault, presets.OpenReads(), "P.md",
		[]ReviewCard{c3, c1, c2},
		map[review.CardID]review.Schedule{c1.ID: {Due: now.Add(-24 * time.Hour), Reps: 1}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !sc.decks["decks/A.md"] || sc.decks["decks/B.md"] || sc.unreviewed != 1 || len(sc.schedules) != 1 {
		t.Fatalf("unexpected scheduledCards: %+v", sc)
	}

	// Test computePlaces error propagation and custom By scheduler.
	if err := pc.computePlaces(3, func(index int) error {
		if index == 1 {
			return errBoom
		}
		return nil
	}); !errors.Is(err, errBoom) {
		t.Fatalf("expected computePlaces to return place error, got %v", err)
	}
	calledBy := false
	pcCustom := pc
	pcCustom.By = func(retention float64) review.Scheduler {
		calledBy = true
		return review.NewFSRSAt(retention)
	}
	_ = pcCustom.getScheduler(0.9)
	if !calledBy {
		t.Fatal("expected custom By scheduler to be called")
	}

	// Test ProjectCurve.Execute error branches (log read error, assignment error, countPointingDecks error, simulation error)
	// and Overdue / Unbegun population.
	rcEnv := NewListReviewCards(env.Ports.Readers, env.Queries, env.Queries)
	schedLogErr := NewSchedules(stubLogStores{openErr: errBoom}, review.NewFSRS(), day, rcEnv, presets)
	if _, err := NewProjectCurve(rcEnv, schedLogErr, presets, day, clock.NewFake(now)).Execute(
		ctx, env.Vault, "P.md", review.Defaults(),
	); !errors.Is(err, errBoom) {
		t.Fatalf("expected log error in ProjectCurve.Execute, got %v", err)
	}

	// Assignment error when path == "" (so getScheduledDecks doesn't use links) and presets.links fails on GetForDeck.
	presetsLinkFail := NewPresets(env.Ports.Readers, env.Ports.Writers, stubLinkQueries{linksErr: errBoom}, env.Queries, scanOf(env), clock.NewFake(now))
	schedLinkFail := NewSchedules(env.Ports.Logs, review.NewFSRS(), day, rcEnv, presetsLinkFail)
	if _, err := NewProjectCurve(rcEnv, schedLinkFail, presetsLinkFail, day, clock.NewFake(now)).Execute(
		ctx, env.Vault, "", review.Defaults(),
	); !errors.Is(err, errBoom) {
		t.Fatalf("expected assignment error in ProjectCurve.Execute, got %v", err)
	}

	// countPointingDecks error when an empty deck (no reviewCards) fails GetForDeck.
	envEmptyDeck := vaulttest.New(t, map[string]string{
		"decks/Empty.md": "---\ntype: deck\n---\n",
	})
	rcEmptyDeck := NewListReviewCards(envEmptyDeck.Ports.Readers, envEmptyDeck.Queries, envEmptyDeck.Queries)
	presetsEmptyFail := NewPresets(
		envEmptyDeck.Ports.Readers, envEmptyDeck.Ports.Writers,
		stubLinkQueries{linksErr: errBoom}, envEmptyDeck.Queries, scanOf(envEmptyDeck), clock.NewFake(now),
	)
	schedEmptyDeck := NewSchedules(envEmptyDeck.Ports.Logs, review.NewFSRS(), day, rcEmptyDeck, presetsEmptyFail)
	if _, err := NewProjectCurve(rcEmptyDeck, schedEmptyDeck, presetsEmptyFail, day, clock.NewFake(now)).Execute(
		ctx, envEmptyDeck.Vault, "", review.Defaults(),
	); !errors.Is(err, errBoom) {
		t.Fatalf("expected countPointingDecks error in ProjectCurve.Execute, got %v", err)
	}

	// Canceled context triggers run.Curve error at line 155.
	ctxCancel, cancel := context.WithCancel(ctx)
	cancel()
	schedEnv := NewSchedules(env.Ports.Logs, review.NewFSRS(), day, rcEnv, presets)
	if _, err := NewProjectCurve(rcEnv, schedEnv, presets, day, clock.NewFake(now)).Execute(
		ctxCancel, env.Vault, "P.md", review.Defaults(),
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled from run.Curve in ProjectCurve.Execute, got %v", err)
	}

	// Successful Execute with 1 overdue card and 1 unbegun card in decks/A.md.
	writer, err := Log{Stores: env.Ports.Logs}.Open(ctx, env.Vault, now.Add(-48*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewRecord(clock.NewFake(now.Add(-48*time.Hour))).Answer(
		ctx, writer, review.CardID{Card: "k7m2xq9fzp", Template: "Card"}, review.Again, time.Second,
	); err != nil {
		t.Fatal(err)
	}
	gotCurve, err := NewProjectCurve(rcEnv, schedEnv, presets, day, clock.NewFake(now)).Execute(
		ctx, env.Vault, "P.md", review.Defaults(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if gotCurve.Overdue != 1 || gotCurve.Unbegun != 1 {
		t.Fatalf("expected Overdue=1 and Unbegun=1, got Overdue=%d Unbegun=%d", gotCurve.Overdue, gotCurve.Unbegun)
	}
}

func TestLogInternalsAndErrors(t *testing.T) {
	ctx := t.Context()
	v := domain.Vault{ID: "v1"}
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	errBoom := errors.New("boom")

	a := review.Answer{
		ID:         "01J00000000000000000000001",
		ReviewCard: review.CardID{Card: "c1", Template: "T"},
		At:         now,
		Rating:     review.Good,
	}
	rl := ReviewLog{Answers: []review.Answer{a}}
	if len(rl.GetHistory()) != 1 {
		t.Fatalf("expected GetHistory with nil order to order answers, got %+v", rl.GetHistory())
	}

	logFail := Log{Stores: stubLogStores{openErr: errBoom}}
	if _, err := logFail.GetLogFiles(ctx, v); !errors.Is(err, errBoom) {
		t.Fatalf("expected Open error in GetLogFiles, got %v", err)
	}
	if _, err := logFail.Read(ctx, v); !errors.Is(err, errBoom) {
		t.Fatalf("expected Open error in Read, got %v", err)
	}
	if _, err := logFail.Open(ctx, v, now); !errors.Is(err, errBoom) {
		t.Fatalf("expected Open error in Log.Open, got %v", err)
	}

	storeGone := stubLogStore{
		readFn: func(_ context.Context, _ string) ([]byte, error) {
			return nil, fs.ErrNotExist
		},
	}
	lfGone, err := (Log{}).ReadFile(ctx, storeGone, ports.Entry{Run: "gone"})
	if err != nil || !lfGone.IsGone || lfGone.Size != 0 || lfGone.Skipped != 0 {
		t.Fatalf("expected IsGone with Size=0 Skipped=0, got %+v, %v", lfGone, err)
	}

	storeLocked := stubLogStore{
		readFn: func(_ context.Context, _ string) ([]byte, error) {
			return nil, ports.ErrLockedByAnother
		},
	}
	lfLocked, err := (Log{}).ReadFile(ctx, storeLocked, ports.Entry{Run: "locked"})
	if err != nil || !lfLocked.IsUnreadable || lfLocked.Size != 0 || lfLocked.Skipped != 1 {
		t.Fatalf("expected IsUnreadable with Size=0 Skipped=1, got %+v, %v", lfLocked, err)
	}

	rec := NewRecord(clock.NewFake(now))
	writer := &LogWriter{store: stubLogStore{}, run: "r1"}
	if _, err := rec.Answer(ctx, writer, review.CardID{Card: "c1", Template: "T"}, review.Rating(99), 0); !errors.Is(err, ErrNoRating) {
		t.Fatalf("expected wrapped ErrNoRating, got %v", err)
	}
	undone, err := rec.TakeBack(ctx, writer, "01J00000000000000000000001")
	if err != nil {
		t.Fatal(err)
	}
	if undone.Undoes != "01J00000000000000000000001" || undone.Rating != 0 || undone.Took != 0 || undone.ReviewCard != (review.CardID{}) {
		t.Fatalf("unexpected TakeBack answer: %+v", undone)
	}

	writerFail := &LogWriter{store: stubLogStore{appendErr: errBoom}, run: "r2"}
	if _, err := rec.TakeBack(ctx, writerFail, "01J00000000000000000000001"); !errors.Is(err, errBoom) {
		t.Fatalf("expected TakeBack append error, got %v", err)
	}
}

func TestNeighborhoodInternalsAndErrors(t *testing.T) {
	ctx := t.Context()
	v := domain.Vault{ID: "v1"}
	errBoom := errors.New("boom")

	if MostRead != 30 {
		t.Fatalf("expected MostRead == 30, got %d", MostRead)
	}

	// Test ShowNeighborhood.Execute error branches: linked error, GetNotes error, GetTypes error,
	// boundary len(out.Notes) == MostRead vs MostRead+1.
	snLinkErr := NewShowNeighborhood(
		note.NewShowLinks(stubLinkQueries{linksErr: errBoom}),
		stubNoteQueries{},
		note.NewRead(stubVaultReaders{}),
	)
	if _, err := snLinkErr.Execute(ctx, v, "decks/A.md"); !errors.Is(err, errBoom) {
		t.Fatalf("expected linked error, got %v", err)
	}

	linksOne := stubLinkQueries{
		links: map[string][]domain.ResolvedLink{
			"decks/A.md": {
				{Link: domain.Link{Target: domain.Address{Scheme: domain.SchemeNote, Value: "Note1.md"}}, To: "Note1.md", ToVault: v.ID},
			},
		},
	}
	snNotesErr := NewShowNeighborhood(
		note.NewShowLinks(linksOne),
		stubNoteQueries{notesErr: errBoom},
		note.NewRead(stubVaultReaders{}),
	)
	if _, err := snNotesErr.Execute(ctx, v, "decks/A.md"); !errors.Is(err, errBoom) {
		t.Fatalf("expected GetNotes error, got %v", err)
	}

	snTypesErr := NewShowNeighborhood(
		note.NewShowLinks(linksOne),
		stubNoteQueries{typesErr: errBoom},
		note.NewRead(stubVaultReaders{}),
	)
	if _, err := snTypesErr.Execute(ctx, v, "decks/A.md"); !errors.Is(err, errBoom) {
		t.Fatalf("expected GetTypes error, got %v", err)
	}

	// Test exactly MostRead (30) notes -> Unread == 0, and readBodies skipping empty Path before reading later note.
	var thirtyLinks []domain.ResolvedLink
	notesMap := make(map[string]domain.NoteRef, 30)
	typesMap := make(map[string]domain.NoteType, 30)
	thirtyLinks = append(thirtyLinks, domain.ResolvedLink{
		Link: domain.Link{Target: domain.Address{Scheme: domain.SchemeName, Value: "Dangling"}},
		To:   "",
	})
	for i := 1; i < 30; i++ {
		p := "Note" + string(rune('A'+i)) + ".md"
		thirtyLinks = append(thirtyLinks, domain.ResolvedLink{
			Link:    domain.Link{Target: domain.Address{Scheme: domain.SchemeNote, Value: p}},
			To:      p,
			ToVault: v.ID,
		})
		notesMap[p] = domain.NoteRef{Path: p, Title: p}
	}
	readerOk := stubVaultReader{
		statFn: func(_ context.Context, path string) (domain.FileStat, error) {
			return domain.FileStat{Path: path, Kind: domain.KindNote, Size: 5}, nil
		},
		readFn: func(_ context.Context, _ string) ([]byte, error) {
			return []byte("prose"), nil
		},
	}
	snThirty := NewShowNeighborhood(
		note.NewShowLinks(stubLinkQueries{links: map[string][]domain.ResolvedLink{"decks/A.md": thirtyLinks}}),
		stubNoteQueries{notes: notesMap, types: typesMap},
		note.NewRead(stubVaultReaders{reader: readerOk}),
	)
	gotThirty, err := snThirty.Execute(ctx, v, "decks/A.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(gotThirty.Notes) != 30 || gotThirty.Unread != 0 {
		t.Fatalf("expected 30 notes and Unread=0, got len=%d Unread=%d", len(gotThirty.Notes), gotThirty.Unread)
	}
	if gotThirty.Notes[1].Body != "prose" {
		t.Fatalf("expected second neighbor body read after dangling first neighbor, got %q", gotThirty.Notes[1].Body)
	}

	// Test getNeighbors duplicate backlinks & seen link, scheme Note vs Name vs http, and getNeighborTitle missing note.
	neighbors := getNeighbors(v, "decks/A.md", note.Links{
		Links: []domain.ResolvedLink{
			{Link: domain.Link{Target: domain.Address{Scheme: "https", Value: "example.com"}}},
			{Link: domain.Link{Target: domain.Address{Scheme: domain.SchemeNote, Value: "N1.md"}}, To: "N1.md", ToVault: v.ID},
			{Link: domain.Link{Target: domain.Address{Scheme: domain.SchemeNote, Value: "N1.md"}}, To: "N1.md", ToVault: v.ID},
		},
		Backlinks: []domain.ResolvedLink{
			{From: "N1.md"},
			{Link: domain.Link{Label: "first"}, From: "N2.md"},
			{Link: domain.Link{Label: "duplicate"}, From: "N2.md"},
			{Link: domain.Link{Label: "after"}, From: "N3.md"},
		},
	})
	if len(neighbors) != 3 || neighbors[0].Path != "N1.md" || neighbors[1].Path != "N2.md" || neighbors[1].Label != "first" || neighbors[2].Path != "N3.md" {
		t.Fatalf("unexpected getNeighbors deduplication: %+v", neighbors)
	}
	if _, ok := getNeighborTitle(Neighbor{Path: "Gone.md"}, map[string]domain.NoteRef{}, map[string]domain.NoteType{}); ok {
		t.Fatal("expected missing note in index to not be included")
	}

	// Test AssignCardIDs: WithIndex, nil notes, GetPathsOfType error, deck read error followed by valid deck.
	indexed := false
	ac := NewAssignCardIDs(nil, nil, nil, nil, nil, nil).WithIndex(func(_ context.Context, _ domain.Vault, _ []string) error {
		indexed = true
		return nil
	})
	if err := ac.index(ctx, v, nil); err != nil || !indexed {
		t.Fatal("expected WithIndex to set index callback")
	}
	resNil, err := ac.Execute(ctx, v)
	if err != nil || len(resNil.Unwritten) != 0 {
		t.Fatalf("expected empty result when notes is nil, got %+v, %v", resNil, err)
	}
	acErr := NewAssignCardIDs(nil, nil, stubNoteQueries{pathsErr: map[domain.NoteType]error{format.TypeDeck: errBoom}}, nil, nil, nil)
	if _, err := acErr.Execute(ctx, v); !errors.Is(err, errBoom) {
		t.Fatalf("expected GetPathsOfType error from AssignCardIDs, got %v", err)
	}

	env := vaulttest.New(t, map[string]string{
		"NotADeck.md":   "plain note\n",
		"decks/Good.md": "---\ntype: deck\n---\n\n## One\n\n[[Term]]\n\n### Front\n\nhi\n",
	})
	acMulti := NewAssignCardIDs(
		env.Ports.Readers, env.Ports.Writers,
		stubNoteQueries{
			paths: map[domain.NoteType][]string{
				format.TypeDeck: {"Missing.md", "NotADeck.md", "decks/Good.md"},
			},
		},
		env.Queries, scanOf(env), clock.NewFake(time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)),
	)
	if _, err := acMulti.Execute(ctx, env.Vault); err != nil {
		t.Fatal(err)
	}
	goodRaw := readVaultFile(t, env, "decks/Good.md")
	if !strings.Contains(goodRaw, "^") {
		t.Fatalf("expected decks/Good.md to receive card ID after skipping unreadable/non-deck notes, got %q", goodRaw)
	}
}

func TestPresetsInternalsAndErrors(t *testing.T) {
	ctx := t.Context()
	v := domain.Vault{ID: "v1"}
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	day := review.Day{Starts: 4 * time.Hour, In: time.UTC}
	errBoom := errors.New("boom")

	p0 := NewPresets(nil, nil, nil, nil, nil, clock.NewFake(now))
	if p0.day.Starts != 0 || p0.day.In != nil {
		t.Fatalf("expected zero day on NewPresets, got %+v", p0.day)
	}
	pDay := p0.WithDay(day)
	if pDay.day.Starts != 4*time.Hour {
		t.Fatalf("expected WithDay to set day, got %+v", pDay.day)
	}

	def := CreateDefaultPreset()
	if def.FileStat.Size != 0 {
		t.Fatalf("expected zero FileStat.Size on CreateDefaultPreset, got %d", def.FileStat.Size)
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
	if _, err := NewPresets(nil, nil, nil, stubNoteQueries{
		pathsErr: map[domain.NoteType]error{format.TypePreset: errBoom},
	}, nil, nil).List(ctx, v); !errors.Is(err, errBoom) {
		t.Fatalf("expected GetPathsOfType error in List, got %v", err)
	}
	emptyList, err := NewPresets(nil, nil, nil, stubNoteQueries{
		paths: map[domain.NoteType][]string{format.TypePreset: nil},
	}, nil, nil).List(ctx, v)
	if err != nil || emptyList != nil {
		t.Fatalf("expected nil, nil for empty preset paths, got %v, %v", emptyList, err)
	}
	if _, err := NewPresets(nil, nil, nil, stubNoteQueries{
		paths:    map[domain.NoteType][]string{format.TypePreset: {"P.md"}},
		notesErr: errBoom,
	}, nil, nil).List(ctx, v); !errors.Is(err, errBoom) {
		t.Fatalf("expected GetNotes error in List, got %v", err)
	}

	// Test GetForDeck error branches: links error, roleless problems error, readFirstLink error, multiple links delta.
	if _, err := NewPresets(nil, nil, stubLinkQueries{linksErr: errBoom}, nil, nil, clock.NewFake(now)).GetForDeck(ctx, v, "decks/A.md"); !errors.Is(err, errBoom) {
		t.Fatalf("expected links error in GetForDeck, got %v", err)
	}
	pProbErr := NewPresets(nil, nil, stubLinkQueries{}, nil, nil, clock.NewFake(now))
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
	gotDangling, err := NewPresets(nil, nil, twoDangling, nil, nil, clock.NewFake(now)).GetForDeck(ctx, v, "decks/A.md")
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
	if _, err := NewPresets(stubVaultReaders{openErr: errBoom}, nil, twoResolved, nil, nil, clock.NewFake(now)).GetForDeck(ctx, v, "decks/A.md"); !errors.Is(err, errBoom) {
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
	gotTwoResolved, err := NewPresets(stubVaultReaders{reader: readerPreset}, nil, twoResolved, nil, nil, clock.NewFake(now)).GetForDeck(ctx, v, "decks/A.md")
	if err != nil || len(gotTwoResolved.Problems) != 1 || !strings.Contains(gotTwoResolved.Problems[0], "scheduled by presets/P.md") {
		t.Fatalf("expected delta presets/P.md in multiple-presets problem, got %+v, %v", gotTwoResolved.Problems, err)
	}

	// Test Read / readPreset / parsePresetBytes branches:
	// missing note (Size==0), Open error, Kind != KindNote, Size == MaxBytes vs MaxBytes+1,
	// StatFile ErrNotExist but Read succeeds (!target && err == nil -> NotANote),
	// invalid UTF-8 -> NotText, unreadable frontmatter -> Unreadable.
	if _, err := NewPresets(stubVaultReaders{openErr: errBoom}, nil, nil, nil, nil, clock.NewFake(now)).Read(ctx, v, "P.md"); !errors.Is(err, errBoom) {
		t.Fatalf("expected Open error in Read, got %v", err)
	}
	readerMissing := stubVaultReader{
		statFn: func(_ context.Context, _ string) (domain.FileStat, error) { return domain.FileStat{}, fs.ErrNotExist },
		readFn: func(_ context.Context, _ string) ([]byte, error) { return nil, fs.ErrNotExist },
	}
	gotMissing, err := NewPresets(stubVaultReaders{reader: readerMissing}, nil, nil, nil, nil, clock.NewFake(now)).Read(ctx, v, "P.md")
	if err != nil || gotMissing.Outcome != note.Missing || gotMissing.FileStat.Size != 0 {
		t.Fatalf("expected Missing with Size=0, got %+v, %v", gotMissing, err)
	}
	readerKindNotNote := stubVaultReader{
		statFn: func(_ context.Context, path string) (domain.FileStat, error) {
			return domain.FileStat{Path: path, Kind: "binary", Size: 10}, nil
		},
		readFn: func(_ context.Context, _ string) ([]byte, error) { return nil, nil },
	}
	gotNotNote, err := NewPresets(stubVaultReaders{reader: readerKindNotNote}, nil, nil, nil, nil, clock.NewFake(now)).Read(ctx, v, "P.bin")
	if err != nil || gotNotNote.Outcome != note.NotANote || gotNotNote.FileStat.Path != "P.bin" {
		t.Fatalf("expected NotANote with FileStat set, got %+v, %v", gotNotNote, err)
	}
	readerTooLarge := stubVaultReader{
		statFn: func(_ context.Context, path string) (domain.FileStat, error) {
			return domain.FileStat{Path: path, Kind: domain.KindNote, Size: note.MaxBytes + 1}, nil
		},
		readFn: func(_ context.Context, _ string) ([]byte, error) { return nil, nil },
	}
	gotTooLarge, err := NewPresets(stubVaultReaders{reader: readerTooLarge}, nil, nil, nil, nil, clock.NewFake(now)).Read(ctx, v, "P.md")
	if err != nil || gotTooLarge.Outcome != note.TooLarge {
		t.Fatalf("expected TooLarge for MaxBytes+1, got %+v, %v", gotTooLarge, err)
	}
	readerExactMax := stubVaultReader{
		statFn: func(_ context.Context, path string) (domain.FileStat, error) {
			return domain.FileStat{Path: path, Kind: domain.KindNote, Size: note.MaxBytes}, nil
		},
		readFn: func(_ context.Context, _ string) ([]byte, error) { return []byte("---\ntype: preset\n---\n"), nil },
	}
	gotExactMax, err := NewPresets(stubVaultReaders{reader: readerExactMax}, nil, nil, nil, nil, clock.NewFake(now)).Read(ctx, v, "P.md")
	if err != nil || gotExactMax.Outcome != note.Ok {
		t.Fatalf("expected Ok for exact MaxBytes, got %+v, %v", gotExactMax, err)
	}
	readerGhost := stubVaultReader{
		statFn: func(_ context.Context, _ string) (domain.FileStat, error) { return domain.FileStat{}, fs.ErrNotExist },
		readFn: func(_ context.Context, _ string) ([]byte, error) { return []byte("---\ntype: preset\n---\n"), nil },
	}
	gotGhost, err := NewPresets(stubVaultReaders{reader: readerGhost}, nil, nil, nil, nil, clock.NewFake(now)).Read(ctx, v, "P.md")
	if err != nil || gotGhost.Outcome != note.NotANote {
		t.Fatalf("expected NotANote when StatFile says not exist but Read succeeds, got %+v, %v", gotGhost, err)
	}
	gotNotText, err := parsePresetBytes(PresetContents{Path: "P.md"}, domain.FileStat{}, []byte{0xff, 0xfe})
	if err != nil || gotNotText.Outcome != note.NotText {
		t.Fatalf("expected NotText for invalid UTF-8, got %+v, %v", gotNotText, err)
	}
	gotUnreadable, err := parsePresetBytes(PresetContents{Path: "P.md"}, domain.FileStat{}, []byte("---\n: : bad yaml\n---\n"))
	if err != nil || gotUnreadable.Outcome != note.Unreadable {
		t.Fatalf("expected Unreadable for broken frontmatter, got %+v, %v", gotUnreadable, err)
	}

	// Test getPresetAddress branches: notes == nil, Read error, pos.Outcome != note.Ok.
	if _, err := (Presets{}).getPresetAddress(ctx, v, "P.md"); !errors.Is(err, ErrNoPresets) {
		t.Fatalf("expected ErrNoPresets when notes == nil, got %v", err)
	}
	if _, err := NewPresets(stubVaultReaders{openErr: errBoom}, nil, nil, stubNoteQueries{}, nil, clock.NewFake(now)).getPresetAddress(ctx, v, "P.md"); !errors.Is(err, errBoom) {
		t.Fatalf("expected Read error in getPresetAddress, got %v", err)
	}
	if _, err := NewPresets(stubVaultReaders{reader: readerTooLarge}, nil, nil, stubNoteQueries{}, nil, clock.NewFake(now)).getPresetAddress(ctx, v, "P.md"); !errors.Is(err, ErrNotAPreset) {
		t.Fatalf("expected ErrNotAPreset when Outcome is TooLarge, got %v", err)
	}

	// Test Save / save error branches: Lock error, release error, readers.Open error, StatFile error,
	// Size == MaxBytes vs MaxBytes+1, expectedFp.IsZero(), Read error, writers.Open error.
	validPreset := review.Defaults()
	if _, err := NewPresets(nil, stubVaultWriters{lockErr: errBoom}, nil, nil, nil, nil).Save(ctx, v, "P.md", validPreset, domain.FileStat{}); !errors.Is(err, errBoom) {
		t.Fatalf("expected lock error in Save, got %v", err)
	}
	if _, err := NewPresets(stubVaultReaders{openErr: errBoom}, stubVaultWriters{}, nil, nil, nil, nil).Save(ctx, v, "P.md", validPreset, domain.FileStat{}); !errors.Is(err, errBoom) {
		t.Fatalf("expected readers.Open error in Save, got %v", err)
	}
	if _, err := NewPresets(stubVaultReaders{reader: readerMissing}, stubVaultWriters{}, nil, nil, nil, nil).Save(ctx, v, "P.md", validPreset, domain.FileStat{}); !errors.Is(err, note.ErrNoNote) {
		t.Fatalf("expected ErrNoNote from StatFile in Save, got %v", err)
	}
	if _, err := NewPresets(stubVaultReaders{reader: readerTooLarge}, stubVaultWriters{}, nil, nil, nil, nil).Save(ctx, v, "P.md", validPreset, domain.FileStat{}); !errors.Is(err, note.ErrTooLarge) {
		t.Fatalf("expected ErrTooLarge in Save, got %v", err)
	}
	readerReadErr := stubVaultReader{
		statFn: func(_ context.Context, path string) (domain.FileStat, error) {
			return domain.FileStat{Path: path, Kind: domain.KindNote, Size: note.MaxBytes}, nil
		},
		readFn: func(_ context.Context, _ string) ([]byte, error) { return nil, fs.ErrNotExist },
	}
	if _, err := NewPresets(stubVaultReaders{reader: readerReadErr}, stubVaultWriters{}, nil, nil, nil, nil).Save(ctx, v, "P.md", validPreset, domain.FileStat{}); !errors.Is(err, note.ErrNoNote) {
		t.Fatalf("expected ErrNoNote from Read in Save, got %v", err)
	}
	if _, err := NewPresets(stubVaultReaders{reader: readerExactMax}, stubVaultWriters{openErr: errBoom}, nil, nil, nil, nil).Save(ctx, v, "P.md", validPreset, domain.FileStat{}); !errors.Is(err, errBoom) {
		t.Fatalf("expected writers.Open error in Save, got %v", err)
	}
	var gotExpectedFp domain.FileStat
	writerCheckFp := stubVaultWriter{
		writeFn: func(_ context.Context, path string, data []byte, expected domain.FileStat) (domain.FileStat, error) {
			gotExpectedFp = expected
			return domain.FileStat{Path: path, Kind: domain.KindNote, Size: int64(len(data))}, nil
		},
	}
	if _, err := NewPresets(
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

func TestReviewCardInternalsAndCache(t *testing.T) {
	ctx := t.Context()
	v := domain.Vault{ID: "v1"}
	errBoom := errors.New("boom")

	rc0 := NewListReviewCards(nil, nil, nil)
	if rc0.Cores != 0 || rc0.MaxCards != 0 || rc0.cache == nil || rc0.cache.total != 0 || rc0.cache.epoch != 0 {
		t.Fatalf("unexpected initial ListReviewCards: %+v", rc0)
	}
	if rc0.getCardLimit() != 1<<18 {
		t.Fatalf("expected defaultCachedCards 1<<18, got %d", rc0.getCardLimit())
	}
	rcLimit1 := rc0
	rcLimit1.MaxCards = 1
	if rcLimit1.getCardLimit() != 1 {
		t.Fatalf("expected MaxCards=1 from getCardLimit, got %d", rcLimit1.getCardLimit())
	}

	if _, err := NewListReviewCards(nil, stubNoteQueries{hasErr: errBoom}, nil).GetDeckPaths(ctx, v); !errors.Is(err, errBoom) {
		t.Fatalf("expected Has error in GetDeckPaths, got %v", err)
	}

	// Test nil reviewCardCache methods.
	var nilCache *reviewCardCache
	if nilCache.begin() != 0 {
		t.Fatalf("expected 0 from nilCache.begin()")
	}
	if dc, ok := nilCache.get(deckKey{}); ok || dc.deck.Size != 0 {
		t.Fatalf("expected zero deckCards and false from nilCache.get, got %+v, %v", dc, ok)
	}
	nilCache.put(0, deckKey{}, deckCards{}, 10)
	nilCache.drop(deckKey{})
	nilCache.keep("v1", nil)
	nilCache.invalidate("v1", nil)
	nilCache.invalidateAll()

	// Test non-nil reviewCardCache: begin, get miss, put epoch mismatch, put len == limit vs len > limit,
	// drop, keep across vaults, invalidate epoch++ and vault check, invalidateAll resetting epoch, order, and total.
	c := newCardCache()
	if dc, ok := c.get(deckKey{vault: "v1", path: "miss.md"}); ok || dc.deck.Size != 0 {
		t.Fatalf("expected zero deckCards and false on cache miss, got %+v, %v", dc, ok)
	}
	c.epoch = 3
	if c.begin() != 3 {
		t.Fatalf("expected begin() == 3, got %d", c.begin())
	}
	key1 := deckKey{vault: "v1", path: "decks/A.md"}
	key2 := deckKey{vault: "v2", path: "decks/B.md"}
	card1 := ReviewCard{ID: review.CardID{Card: "c1", Template: "T"}}
	card2 := ReviewCard{ID: review.CardID{Card: "c2", Template: "T"}}

	c.put(2, key1, deckCards{reviewCards: []ReviewCard{card1}}, 10)
	if _, ok := c.get(key1); ok {
		t.Fatal("expected put with stale epoch to be ignored")
	}
	c.put(3, key1, deckCards{reviewCards: []ReviewCard{card1, card2}}, 1)
	if _, ok := c.get(key1); ok {
		t.Fatal("expected put with len(reviewCards) > limit to be ignored")
	}
	c.put(3, key1, deckCards{
		deck:        domain.FileStat{Path: "decks/A.md", Size: 10},
		cardModels:  map[string]domain.FileStat{"Term.md": {Path: "Term.md", Size: 5}},
		reviewCards: []ReviewCard{card1},
	}, 1)
	if _, ok := c.get(key1); !ok {
		t.Fatal("expected put with len(reviewCards) == limit to be stored")
	}
	c.put(3, key2, deckCards{
		deck:        domain.FileStat{Path: "decks/B.md", Size: 10},
		cardModels:  map[string]domain.FileStat{"Term.md": {Path: "Term.md", Size: 5}},
		reviewCards: []ReviewCard{card2},
	}, 2)
	c.keep("v1", []string{"decks/A.md"})
	if _, ok := c.get(key2); !ok {
		t.Fatal("expected keep on v1 to leave v2 entry untouched")
	}
	c.invalidate("v1", []string{"Other.md"})
	if c.epoch != 4 {
		t.Fatalf("expected invalidate to increment epoch to 4, got %d", c.epoch)
	}
	if _, ok := c.get(key2); !ok {
		t.Fatal("expected invalidate on v1 to leave v2 entry untouched")
	}
	c.drop(key1)
	if _, ok := c.get(key1); ok || c.order.Len() != 1 {
		t.Fatalf("expected drop(key1) to remove key1 from decks and order list, order.Len()=%d", c.order.Len())
	}
	c.invalidateAll()
	if c.epoch != 5 || c.total != 0 || c.order.Len() != 0 || len(c.decks) != 0 {
		t.Fatalf("unexpected state after invalidateAll: epoch=%d total=%d order=%d decks=%d",
			c.epoch, c.total, c.order.Len(), len(c.decks))
	}

	if got := getEntry(&list.Element{Value: "not-cached"}); got.key != (deckKey{}) || got.deck.Size != 0 {
		t.Fatalf("expected zero cached from non-cached element, got %+v", got)
	}
	if got := getEntry(&list.Element{Value: (*cached)(nil)}); got.key != (deckKey{}) || got.deck.Size != 0 {
		t.Fatalf("expected zero cached from nil *cached element, got %+v", got)
	}

	// Test isCurrent when len(e.cardModels) != len(cardModels).
	dc := deckCards{
		deck:       domain.FileStat{Path: "decks/A.md", Size: 10},
		cardModels: map[string]domain.FileStat{"M1.md": {Path: "M1.md", Size: 5}},
	}
	if dc.isCurrent(
		domain.FileStat{Path: "decks/A.md", Size: 10},
		map[string]domain.FileStat{
			"M1.md": {Path: "M1.md", Size: 5},
			"M2.md": {Path: "M2.md", Size: 5},
		},
	) {
		t.Fatal("expected isCurrent false when cardModels map has extra entries")
	}

	// Test getCached branches: deck StatFile error drops cache entry, cardModel StatFile error drops cache entry,
	// and !isCurrent drops cache entry.
	statDeckErr := false
	statModelErr := false
	modShift := time.Duration(0)
	readerDynamic := stubVaultReader{
		statFn: func(_ context.Context, path string) (domain.FileStat, error) {
			if path == "decks/A.md" {
				if statDeckErr {
					return domain.FileStat{}, errBoom
				}
				return domain.FileStat{Path: path, Kind: domain.KindNote, Size: 10, ModTime: time.Unix(100, 0).Add(modShift)}, nil
			}
			if statModelErr {
				return domain.FileStat{}, errBoom
			}
			return domain.FileStat{Path: path, Kind: domain.KindNote, Size: 5, ModTime: time.Unix(100, 0)}, nil
		},
	}
	rcDyn := NewListReviewCards(stubVaultReaders{reader: readerDynamic}, nil, nil)
	seedEntry := deckCards{
		deck:        domain.FileStat{Path: "decks/A.md", Kind: domain.KindNote, Size: 10, ModTime: time.Unix(100, 0)},
		cardModels:  map[string]domain.FileStat{"Term.md": {Path: "Term.md", Kind: domain.KindNote, Size: 5, ModTime: time.Unix(100, 0)}},
		reviewCards: []ReviewCard{card1},
	}

	rcDyn.cache.put(0, key1, seedEntry, 10)
	statDeckErr = true
	if got, ok := rcDyn.getCached(ctx, readerDynamic, v, "decks/A.md"); ok || got != nil {
		t.Fatalf("expected false when deck StatFile errors, got %v, %v", got, ok)
	}
	if _, ok := rcDyn.cache.get(key1); ok {
		t.Fatal("expected cache entry dropped on deck StatFile error")
	}

	statDeckErr = false
	rcDyn.cache.put(0, key1, seedEntry, 10)
	statModelErr = true
	if got, ok := rcDyn.getCached(ctx, readerDynamic, v, "decks/A.md"); ok || got != nil {
		t.Fatalf("expected false when cardModel StatFile errors, got %v, %v", got, ok)
	}
	if _, ok := rcDyn.cache.get(key1); ok {
		t.Fatal("expected cache entry dropped on cardModel StatFile error")
	}

	statModelErr = false
	rcDyn.cache.put(0, key1, seedEntry, 10)
	modShift = time.Second
	if got, ok := rcDyn.getCached(ctx, readerDynamic, v, "decks/A.md"); ok || got != nil {
		t.Fatalf("expected false when deck is modified, got %v, %v", got, ok)
	}
	if _, ok := rcDyn.cache.get(key1); ok {
		t.Fatal("expected cache entry dropped when !isCurrent")
	}

	// Test parseDeck error/not-ok, cardModelMemo.get memoization and invalid card model fields,
	// getDeckCards branches (empty CardID, unresolved CardModelLink, !one.isWhole, empty Front/Back),
	// and getSectionName out-of-bounds section index.
	memo := &cardModelMemo{
		read:       cards.NewRead(stubVaultReaders{openErr: errBoom}, nil),
		cardModels: make(map[string]cardModelRead),
	}
	cmRead := memo.get(ctx, v, "BadModel.md")
	if cmRead.isWhole || cmRead.cardModel.Ref.Size != 0 || cmRead.stat.Size != 0 {
		t.Fatalf("expected zero-sized unwhole cardModelRead on error, got %+v", cmRead)
	}
	if gotNil := rc0.parseDeck(ctx, v, "BadDeck.md", 0, memo); gotNil != nil {
		t.Fatalf("expected nil from parseDeck when read.Deck fails, got %v", gotNil)
	}

	env := vaulttest.New(t, map[string]string{
		"NotAModel.md": "---\ntype: preset\n---\n",
		"Model.md": "---\ntype: card-model\nfields:\n  - Front\n---\n" +
			"\n## EmptyFront\n\n### Front\n\n\n### Back\n\n{{Front}}\n" +
			"\n## EmptyBack\n\n### Front\n\n{{Front}}\n\n### Back\n\n\n" +
			"\n## Valid\n\n### Front\n\n{{Front}}\n\n### Back\n\n{{Front}}\n",
	})
	memoEnv := &cardModelMemo{
		read:       cards.NewRead(env.Ports.Readers, env.Queries),
		cardModels: make(map[string]cardModelRead),
	}
	notModelRead := memoEnv.get(ctx, env.Vault, "NotAModel.md")
	if notModelRead.isWhole {
		t.Fatal("expected note of type preset to not be a whole card model")
	}
	validModelRead := memoEnv.get(ctx, env.Vault, "Model.md")
	if !validModelRead.isWhole {
		t.Fatal("expected Model.md to be whole")
	}
	// Verify memoized entry in m.cardModels is returned even if readers change.
	memoEnv.read = cards.NewRead(stubVaultReaders{openErr: errBoom}, nil)
	if cachedModel := memoEnv.get(ctx, env.Vault, "Model.md"); !cachedModel.isWhole {
		t.Fatal("expected memoized Model.md returned from m.cardModels")
	}

	deckContents := cards.DeckContents{
		Path:    "decks/Mix.md",
		Outcome: note.Ok,
		CardModels: map[string]string{
			"BadLink":   "NotAModel.md",
			"ModelLink": "Model.md",
		},
		Body: format.Deck{
			Sections: []format.Section{{Name: "Sec0"}},
			Cards: []format.Card{
				{CardID: "", CardModelLink: "ModelLink", Values: []format.Value{{Field: "Front", Text: "skip-no-id"}}},
				{CardID: "id1", CardModelLink: "UnresolvedLink", Values: []format.Value{{Field: "Front", Text: "skip-unresolved"}}},
				{CardID: "id2", CardModelLink: "BadLink", Values: []format.Value{{Field: "Front", Text: "skip-not-whole"}}},
				{CardID: "id3", CardModelLink: "ModelLink", Section: 0, Values: []format.Value{{Field: "Front", Text: "kept"}}},
			},
		},
	}
	gotCards, usedModels, isWhole := rc0.getDeckCards(ctx, env.Vault, deckContents, memoEnv)
	if isWhole {
		t.Fatal("expected isWhole=false when one card references NotAModel.md")
	}
	if len(gotCards) != 1 || gotCards[0].ID.Card != "id3" || gotCards[0].ID.Template != "Valid" || len(usedModels) != 1 {
		t.Fatalf("expected only id3/Valid card kept, got %+v", gotCards)
	}

	if gotSec := getSectionName(
		format.Deck{Sections: []format.Section{{Name: "Only"}}},
		format.Card{Section: 1},
	); gotSec != "" {
		t.Fatalf("expected empty section name when c.Section == len(d.Sections), got %q", gotSec)
	}
	if gotSec := getSectionName(
		format.Deck{Sections: []format.Section{{Name: "Only"}}},
		format.Card{Section: 0},
	); gotSec != "Only" {
		t.Fatalf("expected Only when c.Section == 0, got %q", gotSec)
	}
}

func TestScenariosSchedulesAndSessionInternals(t *testing.T) {
	ctx := t.Context()
	v := domain.Vault{ID: "v1"}
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	errBoom := errors.New("boom")

	schedCache := &stubScheduleStore{}
	dayCountCache := &stubScheduleStore{}
	probs := stubProblemQueries{}
	sc := NewScenarios(Config{
		Problems:  probs,
		Schedules: schedCache,
		DayCounts: dayCountCache,
		DayStart:  func() time.Duration { return 3 * time.Hour },
		Now:       clock.NewFake(now),
		Cores:     4,
		Scheduler: stubAlgorithm{},
	})
	if sc.ReviewCards.Cores != 4 || sc.Curves.Cores != 4 {
		t.Fatalf("expected Cores=4 propagated to ReviewCards and Curves, got %d, %d",
			sc.ReviewCards.Cores, sc.Curves.Cores)
	}
	if sc.Day.Starts != 3*time.Hour {
		t.Fatalf("expected DayStart 3h, got %v", sc.Day.Starts)
	}
	if sc.Presets.Problems == nil {
		t.Fatal("expected Presets.Problems wired")
	}
	if sc.Schedules.Cache != schedCache || sc.Counted.Cache != dayCountCache {
		t.Fatal("expected Schedules.Cache and Counted.Cache wired")
	}
	if sc.Schedules.At == nil || sc.Curves.By == nil || sc.Schedules.scheduler == nil {
		t.Fatal("expected custom Scheduler wired to Schedules and Curves")
	}

	if keptVersion != 2 {
		t.Fatalf("expected keptVersion == 2, got %d", keptVersion)
	}

	// Test Schedules: getDayStartMark with non-nil In, getAssignment branches (nil links/notes, ErrNotCarried, error),
	// getAssignmentFrom branches (nil reads/links, GetForDeck error, multiple cards in same deck, mark digest),
	// getMark newline separator, Execute / GetFromLog errors, readRemembered branches, remember sorting.
	loc := time.FixedZone("IST", 19800)
	dayLoc := review.Day{Starts: 2 * time.Hour, In: loc}
	sPlain := NewSchedules(stubLogStores{openErr: errBoom}, review.NewFSRS(), dayLoc, ListReviewCards{}, Presets{})
	if !strings.HasSuffix(sPlain.getDayStartMark(), "\tIST") {
		t.Fatalf("expected timezone name in getDayStartMark, got %q", sPlain.getDayStartMark())
	}
	if getMark([]string{"ab", "c"}) == getMark([]string{"a", "bc"}) {
		t.Fatal("expected getMark newline separator to distinguish [ab,c] from [a,bc]")
	}

	assignPlain, err := sPlain.getAssignment(ctx, v)
	if err != nil || assignPlain.mark == "" {
		t.Fatalf("expected plain assignment when links/notes nil, got %+v, %v", assignPlain, err)
	}
	sNotCarried := NewSchedules(
		stubLogStores{}, review.NewFSRS(), dayLoc,
		NewListReviewCards(stubVaultReaders{}, stubNoteQueries{has: false}, nil),
		Presets{links: stubLinkQueries{}},
	)
	if got, err := sNotCarried.getAssignment(ctx, v); err != nil || got.mark == "" {
		t.Fatalf("expected plain assignment on ErrNotCarried, got %+v, %v", got, err)
	}
	sHasErr := NewSchedules(
		stubLogStores{store: stubLogStore{}}, review.NewFSRS(), dayLoc,
		NewListReviewCards(stubVaultReaders{}, stubNoteQueries{hasErr: errBoom}, nil),
		Presets{links: stubLinkQueries{}},
	)
	if _, err := sHasErr.getAssignment(ctx, v); !errors.Is(err, errBoom) {
		t.Fatalf("expected error from getAssignment when reviewCards.Execute fails, got %v", err)
	}
	if _, err := sHasErr.Execute(ctx, v); !errors.Is(err, errBoom) {
		t.Fatalf("expected getAssignment error in Schedules.Execute, got %v", err)
	}
	if _, err := sHasErr.GetFromLog(ctx, v, ReviewLog{}); !errors.Is(err, errBoom) {
		t.Fatalf("expected getAssignment error in Schedules.GetFromLog, got %v", err)
	}
	if _, err := sPlain.Execute(ctx, v); !errors.Is(err, errBoom) {
		t.Fatalf("expected GetLogFiles error in Schedules.Execute, got %v", err)
	}

	secondLogFail := &countingLogStores{failOnCall: 2, err: errBoom, store: stubLogStore{}}
	sReadFail := NewSchedules(secondLogFail, review.NewFSRS(), dayLoc, ListReviewCards{}, Presets{})
	if _, err := sReadFail.Execute(ctx, v); !errors.Is(err, errBoom) {
		t.Fatalf("expected log.Read error in Schedules.Execute, got %v", err)
	}

	if got, err := sPlain.getAssignmentFrom(ctx, v, nil, nil); err != nil || got.mark == "" {
		t.Fatalf("expected plain assignment when reads is nil, got %+v, %v", got, err)
	}
	if got, err := sPlain.getAssignmentFrom(ctx, v, Presets{}.OpenReads(), nil); err != nil || got.mark == "" {
		t.Fatalf("expected plain assignment when reads.links is nil, got %+v, %v", got, err)
	}
	c1 := ReviewCard{ID: review.CardID{Card: "c1", Template: "T"}, Deck: "decks/A.md"}
	c2 := ReviewCard{ID: review.CardID{Card: "c2", Template: "T"}, Deck: "decks/A.md"}
	if _, err := sPlain.getAssignmentFrom(
		ctx, v, Presets{links: stubLinkQueries{linksErr: errBoom}}.OpenReads(), []ReviewCard{c1},
	); !errors.Is(err, errBoom) {
		t.Fatalf("expected GetForDeck error in getAssignmentFrom, got %v", err)
	}
	readsCount := 0
	readerCount := stubVaultReader{
		statFn: func(_ context.Context, path string) (domain.FileStat, error) {
			readsCount++
			return domain.FileStat{Path: path, Kind: domain.KindNote, Size: 20}, nil
		},
		readFn: func(_ context.Context, _ string) ([]byte, error) {
			return []byte("---\ntype: preset\n---\n"), nil
		},
	}
	readsOnce := NewPresets(
		stubVaultReaders{reader: readerCount}, nil,
		stubLinkQueries{
			links: map[string][]domain.ResolvedLink{
				"decks/A.md": {{Link: domain.Link{Type: LinkType, Target: domain.Address{Scheme: domain.SchemeNote, Value: "P.md"}}, To: "P.md"}},
			},
		},
		nil, nil, clock.NewFake(now),
	).OpenReads()
	if _, err := sPlain.getAssignmentFrom(ctx, v, readsOnce, []ReviewCard{c1, c2}); err != nil {
		t.Fatal(err)
	}
	if readsCount != 1 {
		t.Fatalf("expected deck preset resolved once across two cards in same deck, got %d", readsCount)
	}

	// Test readRemembered invalid JSON, invalid Due timestamp, invalid Last timestamp, mismatched file name,
	// and remember sorting by Card then Template.
	sCache := sPlain
	storeMem := &stubScheduleStore{data: make(map[domain.VaultID][]byte)}
	sCache.Cache = storeMem

	storeMem.data[v.ID] = []byte("{invalid-json")
	if _, ok := sCache.readRemembered(ctx, v, nil, "m"); ok {
		t.Fatal("expected false on invalid JSON in schedule cache")
	}
	storeMem.data[v.ID] = []byte(`{"v":2,"by":"m","files":[],"templates":[{"card":"c","template":"t","due":"bad","last":"2026-06-10T00:00:00Z"}]}`)
	if _, ok := sCache.readRemembered(ctx, v, nil, "m"); ok {
		t.Fatal("expected false on invalid Due timestamp in schedule cache")
	}
	storeMem.data[v.ID] = []byte(`{"v":2,"by":"m","files":[],"templates":[{"card":"c","template":"t","due":"2026-06-10T00:00:00Z","last":"bad"}]}`)
	if _, ok := sCache.readRemembered(ctx, v, nil, "m"); ok {
		t.Fatal("expected false on invalid Last timestamp in schedule cache")
	}
	if read([]cachedFile{{Name: "a", Size: 10}}, []ports.Entry{{Run: "b", Size: 10}}) {
		t.Fatal("expected read() false when file names differ")
	}

	sCache.remember(ctx, v, nil, "m", map[review.CardID]review.Schedule{
		{Card: "b", Template: "2"}: {Due: now, Last: now, Reps: 1},
		{Card: "a", Template: "2"}: {Due: now, Last: now, Reps: 1},
		{Card: "a", Template: "1"}: {Due: now, Last: now, Reps: 1},
	})
	rawRemembered := string(storeMem.data[v.ID])
	idxA1 := strings.Index(rawRemembered, `"card":"a","template":"1"`)
	idxA2 := strings.Index(rawRemembered, `"card":"a","template":"2"`)
	idxB2 := strings.Index(rawRemembered, `"card":"b","template":"2"`)
	if idxA1 < 0 || idxA2 < 0 || idxB2 < 0 || !(idxA1 < idxA2 && idxA2 < idxB2) {
		t.Fatalf("expected ReviewCards sorted by Card then Template in cache JSON: %s", rawRemembered)
	}

	// Test Session.Execute error branches (cardIDs error, reviewCards error, log error, assignment error),
	// due + new card count check when scope.IsNamed, queueCards unscheduled zero fields, and getIntervals nil checks.
	if got := getIntervals(nil, nil, c1.ID, review.Schedule{}, now); got != nil {
		t.Fatalf("expected nil from getIntervals when assign == nil, got %v", got)
	}
	if got := getIntervals(func(review.CardID) review.SchedulingPolicy {
		return review.SchedulingPolicy{By: nil}
	}, nil, c1.ID, review.Schedule{}, now); got != nil {
		t.Fatalf("expected nil from getIntervals when policy.By == nil, got %v", got)
	}
	queued := queueCards(
		asking{newCards: []ReviewCard{c1}},
		map[review.CardID]review.Schedule{},
		review.ScheduleBy(review.NewFSRS()),
		review.NewDueByDay(dayLoc),
		now,
	)
	if len(queued) != 1 || queued[0].Schedule != (review.Schedule{}) || len(queued[0].Ahead) != 4 {
		t.Fatalf("expected zero Schedule and 4 intervals for new card in queueCards, got %+v", queued)
	}

	acFail := NewAssignCardIDs(nil, nil, stubNoteQueries{pathsErr: map[domain.NoteType]error{format.TypeDeck: errBoom}}, nil, nil, nil)
	if _, err := NewSession(acFail, ListReviewCards{}, Schedules{}, Presets{}, dayLoc, clock.NewFake(now)).Execute(ctx, v, Scope{}); !errors.Is(err, errBoom) {
		t.Fatalf("expected cardIDs error in Session.Execute, got %v", err)
	}
	rcFail := NewListReviewCards(nil, stubNoteQueries{hasErr: errBoom}, nil)
	if _, err := NewSession(AssignCardIDs{}, rcFail, Schedules{}, Presets{}, dayLoc, clock.NewFake(now)).Execute(ctx, v, Scope{}); !errors.Is(err, errBoom) {
		t.Fatalf("expected reviewCards error in Session.Execute, got %v", err)
	}
	rcEmptyOk := NewListReviewCards(stubVaultReaders{}, stubNoteQueries{has: true}, nil)
	if _, err := NewSession(AssignCardIDs{}, rcEmptyOk, sPlain, Presets{}, dayLoc, clock.NewFake(now)).Execute(ctx, v, Scope{}); !errors.Is(err, errBoom) {
		t.Fatalf("expected log.Read error in Session.Execute, got %v", err)
	}

	env := vaulttest.New(t, map[string]string{
		"Term.md": "---\ntype: card-model\nfields:\n  - Front\n---\n\n## Card\n\n### Front\n\n{{Front}}\n\n### Back\n\n{{Front}}\n",
		"P.md":    "---\ntype: preset\n---\n",
		"decks/A.md": "---\ntype: deck\nlinks:\n  - to: P\n    role: ref\n    type: preset\n---\n" +
			"\n## One ^k7m2xq9fzp\n\n[[Term]]\n\n### Front\n\nhi\n",
	})
	rcEnv := NewListReviewCards(env.Ports.Readers, env.Queries, env.Queries)
	presetsFail := NewPresets(env.Ports.Readers, env.Ports.Writers, stubLinkQueries{linksErr: errBoom}, env.Queries, scanOf(env), clock.NewFake(now))
	schedEnvFail := NewSchedules(env.Ports.Logs, review.NewFSRS(), dayLoc, rcEnv, presetsFail)
	if _, err := NewSession(AssignCardIDs{}, rcEnv, schedEnvFail, presetsFail, dayLoc, clock.NewFake(now)).Execute(ctx, env.Vault, Scope{}); !errors.Is(err, errBoom) {
		t.Fatalf("expected assignment error in Session.Execute, got %v", err)
	}

	// Test Session.Execute with Scope{Preset: "P.md", IsNamed: true} when len(dueCards)==1 and len(newCards)==1
	// so len(dueCards)+len(newCards) != 0 while len(dueCards)-len(newCards) == 0!
	envTwo := vaulttest.New(t, map[string]string{
		"Term.md": "---\ntype: card-model\nfields:\n  - Front\n---\n\n## Card\n\n### Front\n\n{{Front}}\n\n### Back\n\n{{Front}}\n",
		"P.md":    "---\ntype: preset\n---\n",
		"decks/A.md": "---\ntype: deck\nlinks:\n  - to: P\n    role: ref\n    type: preset\n---\n" +
			"\n## One ^k7m2xq9fzp\n\n[[Term]]\n\n### Front\n\none\n" +
			"\n## Two ^3f4g5h6j7k\n\n[[Term]]\n\n### Front\n\ntwo\n",
	})
	wTwo, err := Log{Stores: envTwo.Ports.Logs}.Open(ctx, envTwo.Vault, now.Add(-48*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewRecord(clock.NewFake(now.Add(-48*time.Hour))).Answer(
		ctx, wTwo, review.CardID{Card: "k7m2xq9fzp", Template: "Card"}, review.Again, time.Second,
	); err != nil {
		t.Fatal(err)
	}
	rcTwo := NewListReviewCards(envTwo.Ports.Readers, envTwo.Queries, envTwo.Queries)
	presetsTwo := NewPresets(envTwo.Ports.Readers, envTwo.Ports.Writers, envTwo.Queries, envTwo.Queries, scanOf(envTwo), clock.NewFake(now)).WithDay(dayLoc)
	schedTwo := NewSchedules(envTwo.Ports.Logs, review.NewFSRS(), dayLoc, rcTwo, presetsTwo)
	sessTwo := NewSession(AssignCardIDs{}, rcTwo, schedTwo, presetsTwo, dayLoc, clock.NewFake(now))
	resTwo, err := sessTwo.Execute(ctx, envTwo.Vault, Scope{Preset: "P.md", IsNamed: true})
	if err != nil || len(resTwo.Queue) != 2 {
		t.Fatalf("expected 2 cards (1 due + 1 new) for named preset session, got %+v, %v", resTwo, err)
	}
}

func scanOf(env vaulttest.Env) func(context.Context, domain.Vault, []string) error {
	return func(context.Context, domain.Vault, []string) error {
		env.Host.Reindex()
		return nil
	}
}

func readVaultFile(t *testing.T, env vaulttest.Env, path string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(env.Vault.Path, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func newSchedulerAt(retention float64) review.Scheduler { return review.NewFSRSAt(retention) }
