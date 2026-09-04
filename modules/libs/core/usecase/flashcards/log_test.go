package flashcards_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// A run is listed and then taken away by another machine's synchroniser before
// it can be read. That is a file gone, not a vault whose history cannot be
// read: everything else the person answered is still theirs.
func TestARunTakenAwayBeforeItWasReadIsGone(t *testing.T) {
	t.Parallel()
	s := opened(t, vault)
	store, err := s.logs.Open(s.vault)
	if err != nil {
		t.Fatal(err)
	}

	ran, err := flashcards.Log{Stores: s.logs}.Run(t.Context(), store, port.Entry{
		Name: "flashcards/01ARZ3NDEKTSV4RRFFQ69G5FAV.jsonl",
	})
	if err != nil {
		t.Fatalf("a run that is no longer there was trouble: %v", err)
	}
	if !ran.Gone {
		t.Error("a run that is no longer there did not say so")
	}
	if len(ran.Answers) != 0 {
		t.Errorf("a run that is no longer there held %d answers", len(ran.Answers))
	}
}

// What a vault holds is every answer of every run, and the runs it was read
// from at the length they were read at — which is what a cache is measured
// against.
func TestWhatAVaultHoldsIsEveryRunItWasReadFrom(t *testing.T) {
	t.Parallel()
	s := opened(t, vault)
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

	held, err := flashcards.Log{Stores: s.logs}.Read(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(held.Answers) != 2 {
		t.Errorf("the vault holds %d answers, want the two written", len(held.Answers))
	}
	if len(held.Files) != 2 {
		t.Errorf("read from %d runs, want the two sittings", len(held.Files))
	}
	for _, one := range held.Files {
		if one.Size == 0 {
			t.Errorf("%s was read at no length at all", one.Name)
		}
	}
	if held.Skipped != 0 {
		t.Errorf("%d lines could not be acted on, want none", held.Skipped)
	}
}

// phantom is a vault's store that lists one run that is not there. It is a
// synchroniser taking a file away between the listing and the reading, which is
// the one moment nothing else can arrange.
type phantom struct {
	port.DerivedStores
	name string
}

func (p phantom) Open(v domain.Vault) (port.DerivedStore, error) {
	store, err := p.DerivedStores.Open(v)
	if err != nil {
		return nil, err
	}
	return listing{DerivedStore: store, name: p.name}, nil
}

type listing struct {
	port.DerivedStore
	name string
}

func (l listing) List(ctx context.Context, name string) ([]port.Entry, error) {
	held, err := l.DerivedStore.List(ctx, name)
	if err != nil {
		return nil, err
	}
	return append(held, port.Entry{Name: l.name, Size: 120}), nil
}

// A run taken away between the listing and the reading is left out, and the
// runs that are still there are read. A vault's history is not refused because
// another machine tidied up while this one was reading.
func TestARunTakenAwayIsLeftOutAndTheRestAreRead(t *testing.T) {
	t.Parallel()
	s := opened(t, vault)
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	if _, err := s.run(t, time.Now()).Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}

	gone := phantom{DerivedStores: s.logs, name: "flashcards/01ARZ3NDEKTSV4RRFFQ69G5FAV.jsonl"}
	held, err := flashcards.Log{Stores: gone}.Read(t.Context(), s.vault)
	if err != nil {
		t.Fatalf("a run taken away refused the whole history: %v", err)
	}
	if len(held.Answers) != 1 {
		t.Errorf("the vault holds %d answers, want the one written", len(held.Answers))
	}
	if len(held.Files) != 1 {
		t.Errorf("read from %d runs, want the one that is there", len(held.Files))
	}
}

// refusing is a vault's store that cannot be read at all: a disk that has gone
// away, a folder somebody's permissions closed.
type refusing struct {
	port.DerivedStores
}

func (r refusing) Open(v domain.Vault) (port.DerivedStore, error) {
	store, err := r.DerivedStores.Open(v)
	if err != nil {
		return nil, err
	}
	return closed{DerivedStore: store}, nil
}

type closed struct{ port.DerivedStore }

var errClosed = errors.New("the folder cannot be read")

func (closed) Read(context.Context, string) ([]byte, error) { return nil, errClosed }

// A vault whose answers cannot be read is not a vault of no answers. The
// difference is a person's whole history, so it is refused and said rather than
// counted as nothing.
func TestAVaultWhoseAnswersCannotBeReadIsRefused(t *testing.T) {
	t.Parallel()
	s := opened(t, vault)
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	if _, err := s.run(t, time.Now()).Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}

	if _, err := (flashcards.Log{Stores: refusing{DerivedStores: s.logs}}).Read(
		t.Context(), s.vault,
	); !errors.Is(err, errClosed) {
		t.Errorf("a folder that cannot be read came back with %v", err)
	}

	counting := s.counted
	counting.Logs = refusing{DerivedStores: s.logs}
	if _, err := counting.Execute(t.Context(), s.vault); !errors.Is(err, errClosed) {
		t.Errorf("the counting came back with %v", err)
	}
}

// A vault folder that is gone mid-sitting is not somewhere to go on answering
// into. An unmounted disk and a sync folder that vanished leave a path the
// application would fill with a stub, and an evening of answers in it is
// shadowed the moment the real vault comes back.
func TestASittingIntoAVaultThatIsGoneStops(t *testing.T) {
	t.Parallel()
	s := opened(t, vault)
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
	if _, err := log.Open(t.Context(), s.vault, time.Now()); err == nil {
		t.Error("a sitting opened on a vault that is gone")
	}
	if _, err := log.Read(t.Context(), s.vault); err == nil {
		t.Error("the history of a vault that is gone was read as no history at all")
	}
}

// brimming is a store that takes one append and refuses every one after it,
// which is a disk filling up under a sitting.
type brimming struct {
	port.DerivedStores
	store *filling
}

func (b *brimming) Open(v domain.Vault) (port.DerivedStore, error) {
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
	port.DerivedStore
	asked int
}

var errNoRoom = errors.New("no room left on the disk")

func (f *filling) Append(ctx context.Context, name string, content []byte) error {
	f.asked++
	if f.asked > 1 {
		return errNoRoom
	}
	return f.DerivedStore.Append(ctx, name, content)
}

// A run whose append did not land stops. The file it was writing ends where a
// line ends, and going on would put the next answer behind whatever landed.
func TestARunWhoseAppendDidNotLandStops(t *testing.T) {
	t.Parallel()
	s := opened(t, vault)
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}

	full := &brimming{DerivedStores: s.logs}
	run, err := flashcards.Log{Stores: full}.Open(t.Context(), s.vault, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	writing := flashcards.Record{Run: run, Now: time.Now}

	if _, err := writing.Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := writing.Answer(t.Context(), on, review.Good, 0); !errors.Is(err, errNoRoom) {
		t.Fatalf("an answer that did not land came back with %v", err)
	}
	if _, err := writing.Answer(t.Context(), on, review.Good, 0); !errors.Is(err, errNoRoom) {
		t.Errorf("the answer after it came back with %v", err)
	}
	if full.store.asked != 2 {
		t.Errorf("the run wrote to the file %d times, want it to stop at the one that did not land",
			full.store.asked)
	}

	held, err := flashcards.Log{Stores: s.logs}.Read(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(held.Answers) != 1 || held.Skipped != 0 {
		t.Errorf("the vault holds %d answers and %d lines it could not act on, want the one that landed",
			len(held.Answers), held.Skipped)
	}
}

// A run file that cannot be read is one run, not the whole window. The reader
// already skips a torn line and counts it, and a file nobody may open is the
// same kind of event: everything else the person answered is returned.
func TestARunThatCannotBeOpenedIsCountedAndTheRestAreRead(t *testing.T) {
	t.Parallel()
	s := opened(t, vault)
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}

	shut := s.run(t, time.Now().AddDate(0, 0, -1))
	if _, err := shut.Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.run(t, time.Now()).Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}
	closed := filepath.Join(s.vault.Path, filesystem.DefaultServiceDir,
		filepath.FromSlash(shut.Run.Name()))
	testsupport.Shut(t, closed)

	held, err := flashcards.Log{Stores: s.logs}.Read(t.Context(), s.vault)
	if err != nil {
		t.Fatalf("one file nobody may open refused the whole history: %v", err)
	}
	if len(held.Answers) != 1 {
		t.Errorf("the vault holds %d answers, want the one that could be read", len(held.Answers))
	}
	if len(held.Files) != 1 {
		t.Errorf("read from %d runs, want the one that could be read", len(held.Files))
	}
	if held.Skipped != 1 {
		t.Errorf("%d could not be acted on, want the one file that could not be opened",
			held.Skipped)
	}

	if _, err := s.session(today).Execute(t.Context(), s.vault, flashcards.Over{}); err != nil {
		t.Errorf("starting a sitting came back with %v", err)
	}
	if _, err := s.counted.Execute(t.Context(), s.vault); err != nil {
		t.Errorf("the counting came back with %v", err)
	}
}

// A vault nobody has reviewed holds no folder and no files, which is an answer
// and not a failure.
func TestAVaultNobodyReviewedHoldsNoRuns(t *testing.T) {
	t.Parallel()
	s := opened(t, vault)

	held, err := flashcards.Log{Stores: s.logs}.Read(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(held.Answers) != 0 || len(held.Files) != 0 {
		t.Errorf("read %+v", held)
	}
}
