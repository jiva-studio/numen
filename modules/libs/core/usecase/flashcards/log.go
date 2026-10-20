package flashcards

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/libs/notes/ulid"
)

// Log is one vault's answers: every run that was ever written there.
type Log struct{ Stores port.DerivedStores }

// ReviewLog is what a vault's log came to.
type ReviewLog struct {
	Answers []review.Answer
	// order is what History hands out, worked out at the first asking and kept
	// for the rest of them.
	order func() review.History
	// Files are what the answers were read from, sorted by name. What tells a
	// cache it is out of date is any difference in this list.
	//
	// A run is appended to and never rewritten, so its length is what says
	// whether it has changed. A name alone says nothing: the file a session is
	// writing to keeps its name and grows all evening.
	Files []port.Entry
	// Skipped is how many lines could not be acted on: a run that stopped
	// partway, or a line of a version this build does not know.
	Skipped int
}

// GetHistory returns the answers in the order they were given: nothing a line takes
// back, one line to an identifier, earliest first.
//
// One request asks several things of one loaded log, and each of them reads this
// order. It is worked out once for the log and handed to all of them.
func (h ReviewLog) GetHistory() review.History {
	if h.order == nil {
		return review.OrderAnswers(h.Answers)
	}
	return h.order()
}

// newHistory is a function that works its order out at the first asking.
func newHistory(answers []review.Answer) func() review.History {
	return sync.OnceValue(func() review.History { return review.OrderAnswers(answers) })
}

// GetLogFiles lists what the vault's log is made of, without reading any of it. It is
// what a cache is measured against, and measuring it costs one listing.
func (u Log) GetLogFiles(ctx context.Context, v domain.Vault) ([]port.Entry, error) {
	store, err := u.Stores.Open(v)
	if err != nil {
		return nil, err
	}
	return readLogFiles(ctx, store)
}

// Read is every answer a vault holds.
//
// A vault nobody has reviewed holds no folder and no files, which is an answer
// and not a failure.
func (u Log) Read(ctx context.Context, v domain.Vault) (ReviewLog, error) {
	store, err := u.Stores.Open(v)
	if err != nil {
		return ReviewLog{}, err
	}
	files, err := readLogFiles(ctx, store)
	if err != nil {
		return ReviewLog{}, err
	}

	var out ReviewLog
	for _, file := range files {
		ran, err := u.ReadFile(ctx, store, file)
		if err != nil {
			return ReviewLog{}, err
		}
		out.Skipped += ran.Skipped
		if ran.IsGone || ran.IsUnreadable {
			continue
		}
		out.Answers = append(out.Answers, ran.Answers...)
		out.Files = append(out.Files, port.Entry{Name: file.Name, Size: ran.Size})
	}
	out.order = newHistory(out.Answers)
	return out, nil
}

// LogFile is one file of the log as it was read.
type LogFile struct {
	Answers []review.Answer
	// Size is the length read, which is what says whether the file has changed.
	// It is the length read and not the length listed: a run this machine is
	// writing grows between the two.
	Size int
	// Skipped is how many of its lines could not be acted on.
	Skipped int
	// IsGone is a file listed and then taken away by another machine's
	// synchroniser before it could be read.
	IsGone bool
	// IsUnreadable is a file that would not open: the permissions refuse it, or
	// another program holds it. It is counted among the lines that could not be
	// acted on and left out of the files the history was read from, so a
	// schedule worked out without it says so.
	IsUnreadable bool
}

// ReadFile reads one file of a vault's log.
func (u Log) ReadFile(
	ctx context.Context, store port.DerivedStore, file port.Entry,
) (LogFile, error) {
	raw, err := store.ReadArtifact(ctx, file.Name)
	if errors.Is(err, fs.ErrNotExist) {
		return LogFile{IsGone: true}, nil
	}
	if errors.Is(err, fs.ErrPermission) || errors.Is(err, port.ErrHeldByAnother) {
		return LogFile{IsUnreadable: true, Skipped: 1}, nil
	}
	if err != nil {
		return LogFile{}, err
	}
	answers, skipped := review.Read(raw)
	return LogFile{Answers: answers, Size: len(raw), Skipped: skipped}, nil
}

// readLogFiles returns the files of the log, sorted by name.
func readLogFiles(ctx context.Context, store port.DerivedStore) ([]port.Entry, error) {
	held, err := store.ListReviewLogs(ctx)
	if err != nil {
		return nil, err
	}
	out := slices.Clone(held)
	slices.SortFunc(out, func(a, b port.Entry) int {
		return strings.Compare(a.Name.Hash, b.Name.Hash)
	})
	return out, nil
}

// Open starts a run.
//
// A run writes one file of its own and nothing else ever appends to it, which
// is what makes two machines' histories merge by being put together: no file is
// ever written by two of them.
func (u Log) Open(ctx context.Context, v domain.Vault, at time.Time) (*LogWriter, error) {
	store, err := u.Stores.Open(v)
	if err != nil {
		return nil, err
	}
	id := ulid.New(at)
	return &LogWriter{store: store, name: domain.NewReviewLogName(id)}, nil
}

// LogWriter is one session of review, and the file it appends to.
type LogWriter struct {
	store port.DerivedStore
	name  domain.ArtifactName
	// stopped is the append that did not land. A run whose file refused one
	// answer writes nothing further to it, and every answer after it is
	// refused with what stopped the first.
	stopped error
}

// GetName is the identifier of the run, which names the file it writes.
func (r *LogWriter) GetName() string { return r.name.Hash }

// Append writes one answer to the end of the run's file.
//
// An append is not atomic: a machine that stopped mid-line leaves a tail no
// newline closes, and reading the file back leaves that line out.
func (r *LogWriter) Append(ctx context.Context, a review.Answer) error {
	if r.stopped != nil {
		return r.stopped
	}
	raw, err := review.Write(a)
	if err != nil {
		return err
	}
	if err := r.store.AppendArtifact(ctx, r.name, raw); err != nil {
		r.stopped = fmt.Errorf("%s took no more answers: %w", r.name, err)
		return r.stopped
	}
	return nil
}
