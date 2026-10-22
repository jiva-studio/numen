package flashcards_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/flashcards"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/ports"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/review"
)

// A run is listed and then taken away by another machine's synchroniser before
// it can be read. That is a file gone, not a vault whose history cannot be
// read: everything else the person answered is still theirs.
func TestARunTakenAwayBeforeItWasReadIsMissing(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	store, err := s.logs.Open(s.vault)
	if err != nil {
		t.Fatal(err)
	}

	ran, err := flashcards.Log{Stores: s.logs}.ReadFile(t.Context(), store, ports.Entry{
		Name: domain.NewReviewLogName("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
	})
	if err != nil {
		t.Fatalf("a run that is no longer there was trouble: %v", err)
	}
	if !ran.IsGone {
		t.Error("a run that is no longer there did not say so")
	}
	if len(ran.Answers) != 0 {
		t.Errorf("a run that is no longer there held %d answers", len(ran.Answers))
	}
}

// What a vault holds is every answer of every run, and the runs it was read
// from at the length they were read at — which is what a cache is measured
// against.
func TestWhatAVaultContainsIsEveryRunItWasReadFrom(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	other := review.CardFaceID{Card: "zpqrstvwxy", Face: "Recognise"}

	if _, err := s.run(t, time.Now().AddDate(0, 0, -1)).Answer(
		t.Context(), on, review.Good, 0,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := s.run(t, time.Now()).Answer(t.Context(), other, review.Good, 0); err != nil {
		t.Fatal(err)
	}

	target, err := flashcards.Log{Stores: s.logs}.Read(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(target.Answers) != 2 {
		t.Errorf("the vault holds %d answers, want the two written", len(target.Answers))
	}
	if len(target.Files) != 2 {
		t.Errorf("read from %d runs, want the two sessions", len(target.Files))
	}
	for _, one := range target.Files {
		if one.Size == 0 {
			t.Errorf("%s was read at no length at all", one.Name)
		}
	}
	if target.Skipped != 0 {
		t.Errorf("%d lines could not be acted on, want none", target.Skipped)
	}
}

// phantom is a vault's store that lists one run that is not there. It is a
// synchroniser taking a file away between the listing and the reading, which is
// the one moment nothing else can arrange.
type phantom struct {
	ports.DerivedStores
	name domain.ArtifactName
}

func (p phantom) Open(v domain.Vault) (ports.DerivedStore, error) {
	store, err := p.DerivedStores.Open(v)
	if err != nil {
		return nil, err
	}
	return listing{DerivedStore: store, name: p.name}, nil
}

type listing struct {
	ports.DerivedStore
	name domain.ArtifactName
}

func (l listing) ListReviewLogs(ctx context.Context) ([]ports.Entry, error) {
	target, err := l.DerivedStore.ListReviewLogs(ctx)
	if err != nil {
		return nil, err
	}
	return append(target, ports.Entry{Name: l.name, Size: 120}), nil
}

// A run taken away between the listing and the reading is left out, and the
// runs that are still there are read. A vault's history is not rejected because
// another machine tidied up while this one was reading.
func TestARunTakenAwayIsLeftOutAndTheRestAreRead(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	if _, err := s.run(t, time.Now()).Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}

	removed := phantom{DerivedStores: s.logs, name: domain.NewReviewLogName("01ARZ3NDEKTSV4RRFFQ69G5FAV")}
	target, err := flashcards.Log{Stores: removed}.Read(t.Context(), s.vault)
	if err != nil {
		t.Fatalf("a run taken away failed the whole history: %v", err)
	}
	if len(target.Answers) != 1 {
		t.Errorf("the vault holds %d answers, want the one written", len(target.Answers))
	}
	if len(target.Files) != 1 {
		t.Errorf("read from %d runs, want the one that is there", len(target.Files))
	}
}

// reordered is a vault's store whose listing flips between ascending and
// descending order each time it is asked, which is what a directory's own
// order is not promised to keep across two listings of it.
type reordered struct {
	ports.DerivedStore
	reverse bool
}

func (r *reordered) ListReviewLogs(ctx context.Context) ([]ports.Entry, error) {
	target, err := r.DerivedStore.ListReviewLogs(ctx)
	if err != nil {
		return nil, err
	}
	out := slices.Clone(target)
	if r.reverse {
		slices.Reverse(out)
	}
	r.reverse = !r.reverse
	return out, nil
}

// reorderedStores opens the one store every call flips the order of, so the
// same vault's listing keeps flipping across every call that asks for it.
type reorderedStores struct {
	ports.DerivedStores
	store *reordered
}

func (s *reorderedStores) Open(v domain.Vault) (ports.DerivedStore, error) {
	if s.store == nil {
		inner, err := s.DerivedStores.Open(v)
		if err != nil {
			return nil, err
		}
		s.store = &reordered{DerivedStore: inner}
	}
	return s.store, nil
}

// GetLogFiles sorts what the store lists, so the same runs come back in the same
// order whatever order the store's own listing came back in.
func TestGetLogFilesSortsWhateverOrderTheStoreLists(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	for _, item := range []time.Time{
		time.Now().AddDate(0, 0, -2),
		time.Now().AddDate(0, 0, -1),
		time.Now(),
	} {
		if _, err := s.run(t, item).Answer(t.Context(), on, review.Good, 0); err != nil {
			t.Fatal(err)
		}
	}

	log := flashcards.Log{Stores: &reorderedStores{DerivedStores: s.logs}}
	first, err := log.GetLogFiles(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	second, err := log.GetLogFiles(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 3 || len(second) != 3 {
		t.Fatalf("listed %d and then %d files, want the three runs each time", len(first), len(second))
	}
	byName := func(a, b ports.Entry) int { return strings.Compare(a.Name.Hash, b.Name.Hash) }
	for _, one := range [][]ports.Entry{first, second} {
		if !slices.IsSortedFunc(one, byName) {
			t.Errorf("%v is not sorted by name", one)
		}
	}
	for entry := range first {
		if first[entry].Name != second[entry].Name {
			t.Errorf("the same runs came back in another order: %v then %v", first, second)
		}
	}
}

// refusing is a vault's store that cannot be read at all: a disk that has gone
// away, a folder somebody's permissions closed.
type refusing struct {
	ports.DerivedStores
}

func (r refusing) Open(v domain.Vault) (ports.DerivedStore, error) {
	store, err := r.DerivedStores.Open(v)
	if err != nil {
		return nil, err
	}
	return closed{DerivedStore: store}, nil
}

type closed struct{ ports.DerivedStore }

var errClosed = errors.New("the folder cannot be read")

func (closed) ReadArtifact(context.Context, domain.ArtifactName) ([]byte, error) {
	return nil, errClosed
}

// A vault whose answers cannot be read is not a vault of no answers. The
// difference is a person's whole history, so it fails and said rather than
// counted as nothing.
func TestAVaultWhoseAnswersCannotBeReadIsRefused(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	if _, err := s.run(t, time.Now()).Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}

	if _, err := (flashcards.Log{Stores: refusing{DerivedStores: s.logs}}).Read(
		t.Context(), s.vault,
	); !errors.Is(err, errClosed) {
		t.Errorf("a folder that cannot be read came back with %v", err)
	}

	counter := flashcards.NewCountReviews(refusing{DerivedStores: s.logs}, s.schedules, today, time.Now)
	if _, err := counter.Execute(t.Context(), s.vault); !errors.Is(err, errClosed) {
		t.Errorf("the counting came back with %v", err)
	}
}

// A vault folder that is gone mid-session is not somewhere to go on answering
// into. An unmounted disk and a sync folder that vanished leave a path the
// application would fill with a stub, and an evening of answers in it is
// shadowed the moment the real vault comes back.
func TestASessionIntoAVaultThatIsMissingStops(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	writing := s.run(t, time.Now())
	if _, err := writing.Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}

	if err := os.RemoveAll(s.vault.Path); err != nil {
		t.Fatal(err)
	}

	if _, err := writing.Answer(t.Context(), on, review.Good, 0); err == nil {
		t.Error("an answer into a folder that is no longer the vault said it landed")
	}
	if _, err := os.Stat(s.vault.Path); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the vault was made again to answer into: %v", err)
	}

	log := flashcards.Log{Stores: s.logs}
	if _, err := log.Read(t.Context(), s.vault); err == nil {
		t.Error("the history of a vault that is gone was read as no history at all")
	}
}

// brimming is a store that takes one append and refuses every one after it,
// which is a disk filling up under a session.
type brimming struct {
	ports.DerivedStores
	store *filling
}

func (b *brimming) Open(v domain.Vault) (ports.DerivedStore, error) {
	if b.store == nil {
		store, err := b.DerivedStores.Open(v)
		if err != nil {
			return nil, err
		}
		b.store = &filling{DerivedStore: store}
	}
	return b.store, nil
}

type filling struct {
	ports.DerivedStore
	callCount int
}

var errNoRoom = errors.New("no room left on the disk")

func (f *filling) AppendArtifact(ctx context.Context, name domain.ArtifactName, content []byte) error {
	f.callCount++
	if f.callCount > 1 {
		return errNoRoom
	}
	return f.DerivedStore.AppendArtifact(ctx, name, content)
}

// A run whose append did not land stops. The file it was writing ends where a
// line ends, and going on would put the next answer behind whatever landed.
func TestARunWhoseAppendDidNotLandStops(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}

	full := &brimming{DerivedStores: s.logs}
	run, err := flashcards.Log{Stores: full}.Open(t.Context(), s.vault, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	writing := newSitting(run, time.Now)

	if _, err := writing.Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := writing.Answer(t.Context(), on, review.Good, 0); !errors.Is(err, errNoRoom) {
		t.Fatalf("an answer that did not land came back with %v", err)
	}
	if _, err := writing.Answer(t.Context(), on, review.Good, 0); !errors.Is(err, errNoRoom) {
		t.Errorf("the answer after it came back with %v", err)
	}
	if full.store.callCount != 2 {
		t.Errorf("the run wrote to the file %d times, want it to stop at the one that did not land",
			full.store.callCount)
	}

	target, err := flashcards.Log{Stores: s.logs}.Read(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(target.Answers) != 1 || target.Skipped != 0 {
		t.Errorf("the vault holds %d answers and %d lines it could not act on, want the one that landed",
			len(target.Answers), target.Skipped)
	}
}

// A run file that cannot be read is one run, not the whole window. The reader
// already skips a torn line and counts it, and a file nobody may open is the
// same kind of event: everything else the person answered is returned.
func TestARunThatCannotBeOpenedIsCountedAndTheRestAreRead(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}

	yesterday := time.Now().AddDate(0, 0, -1)
	yesterdayRun, err := flashcards.Log{Stores: s.logs}.Open(t.Context(), s.vault, yesterday)
	if err != nil {
		t.Fatal(err)
	}
	yesterdaySession := newSitting(yesterdayRun, func() time.Time { return yesterday })
	if _, err := yesterdaySession.Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.run(t, time.Now()).Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}
	s.env.Host.RefuseData(yesterdayRun.GetName()+".jsonl", fs.ErrPermission)

	target, err := flashcards.Log{Stores: s.logs}.Read(t.Context(), s.vault)
	if err != nil {
		t.Fatalf("one file nobody may open failed the whole history: %v", err)
	}
	if len(target.Answers) != 1 {
		t.Errorf("the vault holds %d answers, want the one that could be read", len(target.Answers))
	}
	if len(target.Files) != 1 {
		t.Errorf("read from %d runs, want the one that could be read", len(target.Files))
	}
	if target.Skipped != 1 {
		t.Errorf("%d could not be acted on, want the one file that could not be opened",
			target.Skipped)
	}

	if _, err := s.session(today).Execute(t.Context(), s.vault, flashcards.Scope{}); err != nil {
		t.Errorf("starting a session came back with %v", err)
	}
	if _, err := s.counted.Execute(t.Context(), s.vault); err != nil {
		t.Errorf("the counting came back with %v", err)
	}
}

// A vault nobody has reviewed holds no folder and no files, which is an answer
// and not a failure.
func TestAVaultNobodyReviewedContainsNoRuns(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)

	target, err := flashcards.Log{Stores: s.logs}.Read(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(target.Answers) != 0 || len(target.Files) != 0 {
		t.Errorf("read %+v", target)
	}
}
