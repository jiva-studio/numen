package flashcards

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/ulid"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// Log is one vault's answers: every run that was ever written there. Entropy fills the identifier
// of a run Open starts.
type Log struct {
	Stores  ports.LogStores
	Entropy ports.Entropy
}

// ReviewLog is what a vault's log came to.
type ReviewLog struct {
	Answers []review.Answer
	order   func() review.History
	Files   []ports.Entry
	Skipped int
}

// GetHistory returns the answers in the order they were given: nothing a line takes
// back, one line to an identifier, earliest first.
//
// One request evaluates several things over one loaded log, and each reads this
// order. It is computed once per log and shared.
func (h ReviewLog) GetHistory() review.History {
	if h.order == nil {
		return review.OrderAnswers(h.Answers)
	}
	return h.order()
}

func newHistory(answers []review.Answer) func() review.History {
	return sync.OnceValue(func() review.History { return review.OrderAnswers(answers) })
}

// GetLogFiles lists what the vault's log is made of, without reading any of it. It is
// what a cache is measured against, and measuring it costs one listing.
func (u Log) GetLogFiles(ctx context.Context, v domain.Vault) ([]ports.Entry, error) {
	store, err := u.Stores.Open(v)
	if err != nil {
		return nil, err
	}
	return readLogFiles(ctx, store)
}

// Read is every answer a vault holds.
//
// A vault with no reviews holds no folder and no files, which yields an empty
// log and not an error.
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
		out.Files = append(out.Files, ports.Entry{Run: file.Run, Size: ran.Size})
	}
	out.order = newHistory(out.Answers)
	return out, nil
}

type LogFile struct {
	Answers      []review.Answer
	Size         int
	Skipped      int
	IsGone       bool
	IsUnreadable bool
}

// ReadFile reads one file of a vault's log.
func (u Log) ReadFile(
	ctx context.Context, store ports.LogStore, file ports.Entry,
) (LogFile, error) {
	raw, err := store.ReadLog(ctx, file.Run)
	if errors.Is(err, ports.ErrNotExist) {
		return LogFile{Answers: nil, Size: 0, Skipped: 0, IsGone: true, IsUnreadable: false}, nil
	}
	if errors.Is(err, ports.ErrPermission) || errors.Is(err, ports.ErrLockedByAnother) {
		return LogFile{Answers: nil, Size: 0, IsUnreadable: true, Skipped: 1, IsGone: false}, nil
	}
	if err != nil {
		return LogFile{}, err
	}
	answers, skipped := review.Read(raw)
	return LogFile{Answers: answers, Size: len(raw), Skipped: skipped, IsGone: false, IsUnreadable: false}, nil
}

func readLogFiles(ctx context.Context, store ports.LogStore) ([]ports.Entry, error) {
	target, err := store.ListLogs(ctx)
	if err != nil {
		return nil, err
	}
	out := slices.Clone(target)
	slices.SortFunc(out, func(a, b ports.Entry) int {
		return strings.Compare(a.Run, b.Run)
	})
	return out, nil
}

// Open starts a run.
//
// A run writes one file of its own and nothing else ever appends to it, which
// is what makes two machines' histories merge by being put together: no file is
// ever written by two of them.
func (u Log) Open(_ context.Context, v domain.Vault, now time.Time) (*LogWriter, error) {
	store, err := u.Stores.Open(v)
	if err != nil {
		return nil, err
	}
	id, err := ulid.NewGenerator(format.NewEntropyReader(u.Entropy)).NewID(now)
	// gomutants:disable-next-line BRANCH_IF reason="ULID generation from the entropy of the machine does not fail."
	if err != nil {
		return nil, err
	}
	return &LogWriter{store: store, run: id, stopped: nil}, nil
}

// LogWriter is one session of review, and the file it appends to.
type LogWriter struct {
	store   ports.LogStore
	run     string
	stopped error
}

// GetName is the identifier of the run, which names the file it writes.
func (r *LogWriter) GetName() string { return r.run }

// Append writes one answer to the end of the run's file.
//
// An append is not atomic: a machine that stopped mid-line leaves a tail no
// newline closes, and reading the file back leaves that line out.
func (r *LogWriter) Append(ctx context.Context, a review.Answer) error {
	if r.stopped != nil {
		return r.stopped
	}
	raw, err := review.Write(a)
	// gomutants:disable-next-line BRANCH_IF reason="Marshaling wireAnswer with string and integer fields cannot fail."
	if err != nil {
		return err
	}
	if err := r.store.AppendLog(ctx, r.run, raw); err != nil {
		r.stopped = fmt.Errorf("run %s took no more answers: %w", r.run, err)
		return r.stopped
	}
	return nil
}

// ErrNoRating is returned for an answer outside the four ratings. A rating is
// never inferred.
var ErrNoRating = errors.New("not one of the four ratings")

// ErrNoAnswer is a taking back that names no answer.
var ErrNoAnswer = errors.New("an answer taken back names the one it takes back")

// Record writes answers to the log.
//
// A line is appended and never rewritten, so a recorded answer persists, including
// when it is taken back: taking back is a line of its own naming the answer it
// reverses.
type Record struct {
	now     ports.Clock
	entropy ports.Entropy
}

// errNoClock is what a Record that was given no clock answers.
var errNoClock = errors.New("an answer is stamped by a clock, and none was given")

// NewRecord is what an answer is written down through: what time it is, and the entropy of the
// identifier of each line.
func NewRecord(now ports.Clock, entropy ports.Entropy) Record {
	return Record{now: now, entropy: entropy}
}

// Answer writes down one card answered once, and returns the line as it was
// written: it carries the identifier that taking this answer back would name.
func (u Record) Answer(
	ctx context.Context, run *LogWriter, on review.CardID, r review.Rating, took time.Duration,
) (review.Answer, error) {
	if !r.Valid() {
		return review.Answer{}, fmt.Errorf("%w: %d", ErrNoRating, r)
	}
	if on.Card == "" || on.Template == "" {
		return review.Answer{}, errors.New("an answer requires a card and a review card")
	}

	if u.now == nil {
		return review.Answer{}, errNoClock
	}
	pos := u.now.Now()
	id, err := ulid.NewGenerator(format.NewEntropyReader(u.entropy)).NewID(pos)
	// gomutants:disable-next-line BRANCH_IF reason="ULID generation from the entropy of the machine does not fail."
	if err != nil {
		return review.Answer{}, err
	}
	a := review.Answer{ID: id, ReviewCard: on, At: pos, Rating: r, Took: took, Undoes: ""}
	if err := run.Append(ctx, a); err != nil {
		return review.Answer{}, err
	}
	return a, nil
}

// TakeBack writes that an answer was taken back. The record of the reversal
// marks the answer as reversed.
func (u Record) TakeBack(ctx context.Context, run *LogWriter, id string) (review.Answer, error) {
	if id == "" {
		return review.Answer{}, ErrNoAnswer
	}
	if u.now == nil {
		return review.Answer{}, errNoClock
	}
	pos := u.now.Now()
	own, err := ulid.NewGenerator(format.NewEntropyReader(u.entropy)).NewID(pos)
	// gomutants:disable-next-line BRANCH_IF reason="ULID generation from the entropy of the machine does not fail."
	if err != nil {
		return review.Answer{}, err
	}
	a := review.Answer{ID: own, ReviewCard: review.CardID{Card: "", Template: ""}, At: pos, Rating: review.Rating(0), Took: time.Duration(0), Undoes: id}
	if err := run.Append(ctx, a); err != nil {
		return review.Answer{}, err
	}
	return a, nil
}
