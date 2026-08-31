package flashcards

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/internal/ulid"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Area is the folder inside a vault's service folder the answers are kept in.
const Area = "flashcards"

// Suffix is what a file of answers is named with.
const Suffix = ".jsonl"

// Log is one vault's answers: every run that was ever written there.
type Log struct{ Stores port.DerivedStores }

// Held is what a vault's log came to.
type Held struct {
	Answers []history.Answer
	// Files are what the answers were read from, sorted by name. What tells a
	// cache it is out of date is any difference in this list.
	//
	// A run is appended to and never rewritten, so its length is what says
	// whether it has changed. A name alone says nothing: the file a sitting is
	// writing to keeps its name and grows all evening.
	Files []port.Stored
	// Skipped is how many lines could not be acted on: a run that stopped
	// partway, or a line of a version this build does not know.
	Skipped int
}

// Files is what the vault's log is made of, without reading any of it. It is
// what a cache is measured against, and measuring it costs one listing.
func (u Log) Files(ctx context.Context, v domain.Vault) ([]port.Stored, error) {
	store, err := u.Stores.Open(v)
	if err != nil {
		return nil, err
	}
	return runs(ctx, store)
}

// Read is every answer a vault holds.
//
// A vault nobody has reviewed holds no folder and no files, which is an answer
// and not a failure.
func (u Log) Read(ctx context.Context, v domain.Vault) (Held, error) {
	store, err := u.Stores.Open(v)
	if err != nil {
		return Held{}, err
	}
	files, err := runs(ctx, store)
	if err != nil {
		return Held{}, err
	}

	var out Held
	for _, file := range files {
		ran, err := u.Run(ctx, store, file)
		if err != nil {
			return Held{}, err
		}
		out.Skipped += ran.Skipped
		if ran.Gone || ran.Shut {
			continue
		}
		out.Answers = append(out.Answers, ran.Answers...)
		out.Files = append(out.Files, port.Stored{Name: file.Name, Size: ran.Size})
	}
	return out, nil
}

// Ran is one file of the log as it was read.
type Ran struct {
	Answers []history.Answer
	// Size is the length read, which is what says whether the file has changed.
	// It is the length read and not the length listed: a run this machine is
	// writing grows between the two.
	Size int
	// Skipped is how many of its lines could not be acted on.
	Skipped int
	// Gone is a file listed and then taken away by another machine's
	// synchroniser before it could be read.
	Gone bool
	// Shut is a file the permissions on it keep closed. It is counted among
	// the lines that could not be acted on and left out of the files the
	// history was read from, so a schedule worked out without it says so.
	Shut bool
}

// Run is one file of a vault's log, read.
func (u Log) Run(ctx context.Context, store port.DerivedStore, file port.Stored) (Ran, error) {
	raw, err := store.Read(ctx, file.Name)
	if errors.Is(err, fs.ErrNotExist) {
		return Ran{Gone: true}, nil
	}
	if errors.Is(err, fs.ErrPermission) {
		return Ran{Shut: true, Skipped: 1}, nil
	}
	if err != nil {
		return Ran{}, err
	}
	answers, skipped := history.Read(raw)
	return Ran{Answers: answers, Size: len(raw), Skipped: skipped}, nil
}

// runs is the files of the log, sorted by name.
func runs(ctx context.Context, store port.DerivedStore) ([]port.Stored, error) {
	held, err := store.List(ctx, Area)
	if err != nil {
		return nil, err
	}
	out := make([]port.Stored, 0, len(held))
	for _, one := range held {
		if strings.HasSuffix(one.Name, Suffix) {
			out = append(out, one)
		}
	}
	return out, nil
}

// Open starts a run.
//
// A run writes one file of its own and nothing else ever appends to it, which
// is what makes two machines' histories merge by being put together: no file is
// ever written by two of them.
func (u Log) Open(ctx context.Context, v domain.Vault, at time.Time) (*Run, error) {
	store, err := u.Stores.Open(v)
	if err != nil {
		return nil, err
	}
	id, err := ulid.New(at)
	if err != nil {
		return nil, err
	}
	return &Run{store: store, name: Area + "/" + id + Suffix}, nil
}

// Run is one sitting of review, and the file it appends to.
type Run struct {
	store port.DerivedStore
	name  string
	// stopped is the append that did not land. A run whose file refused one
	// answer writes nothing further to it, and every answer after it is
	// refused with what stopped the first.
	stopped error
}

// Name is the file this run writes, as a name of the vault's own store.
func (r *Run) Name() string { return r.name }

// Append writes one answer to the end of the run's file.
//
// An append is not atomic: a machine that stopped mid-line leaves a tail no
// newline closes, and reading the file back leaves that line out.
func (r *Run) Append(ctx context.Context, a history.Answer) error {
	if r.stopped != nil {
		return r.stopped
	}
	raw, err := history.Write(a)
	if err != nil {
		return err
	}
	if err := r.store.Append(ctx, r.name, raw); err != nil {
		r.stopped = fmt.Errorf("%s took no more answers: %w", r.name, err)
		return r.stopped
	}
	return nil
}
