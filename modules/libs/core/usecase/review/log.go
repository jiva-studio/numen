package review

import (
	"context"
	"errors"
	"io/fs"
	"strings"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/ulid"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	history "github.com/jiva-studio/numen/modules/libs/core/review"
)

// Area is the folder inside a vault's service folder the answers are kept in.
const Area = "review"

// Suffix is what a file of answers is named with.
const Suffix = ".jsonl"

// Log is one vault's answers: every run that was ever written there.
type Log struct{ Stores port.DerivedStores }

// Counted is one file of the log as it was read: its name, and how many bytes
// it held.
//
// A run is appended to and never rewritten, so its length is what says whether
// it has changed. A name alone says nothing: the file a sitting is writing to
// keeps its name and grows all evening.
type Counted struct {
	Name string
	Size int
}

// Held is what a vault's log came to.
type Held struct {
	Answers []history.Answer
	// Files are what the answers were read from, sorted by name. What tells a
	// cache it is out of date is any difference in this list.
	Files []Counted
	// Skipped is how many lines could not be acted on: a run that stopped
	// partway, or a line of a version this build does not know.
	Skipped int
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
	names, err := store.List(ctx, Area)
	if err != nil {
		return Held{}, err
	}

	var out Held
	for _, name := range names {
		if !strings.HasSuffix(name, Suffix) {
			continue
		}
		raw, err := store.Read(ctx, name)
		if errors.Is(err, fs.ErrNotExist) {
			// A file listed and then gone is a file another machine's
			// synchroniser took away while this was reading.
			continue
		}
		if err != nil {
			return Held{}, err
		}
		answers, skipped := history.Read(raw)
		out.Answers = append(out.Answers, answers...)
		out.Files = append(out.Files, Counted{Name: name, Size: len(raw)})
		out.Skipped += skipped
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
}

// Name is the file this run writes, as a name of the vault's own store.
func (r *Run) Name() string { return r.name }

// Append writes one answer to the end of the run's file.
//
// An append is not atomic: a machine that stopped mid-line leaves a tail no
// newline closes, and reading the file back leaves that line out.
func (r *Run) Append(ctx context.Context, a history.Answer) error {
	raw, err := history.Write(a)
	if err != nil {
		return err
	}
	return r.store.Append(ctx, r.name, raw)
}
