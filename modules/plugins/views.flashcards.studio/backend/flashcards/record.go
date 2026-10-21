package flashcards

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/modules/libs/notes/ulid"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/ports"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/review"
)

// ErrNoRating is what an answer outside the four gets. Nothing here guesses
// what a person meant to say about their own recall.
var ErrNoRating = errors.New("not one of the four ratings")

// ErrNoAnswer is a taking back that names no answer.
var ErrNoAnswer = errors.New("an answer taken back names the one it takes back")

// Record writes down what a person answered.
//
// A line is appended and nothing is ever rewritten, so what was recorded stands
// whatever happens next — including a person taking an answer back, which is a
// line of its own naming the one it takes back.
type Record struct{ now ports.Clock }

// errNoClock is what a Record that was given no clock answers.
var errNoClock = errors.New("an answer is stamped by a clock, and none was given")

// NewRecord is what an answer is written down through: what time it is.
func NewRecord(now ports.Clock) Record { return Record{now: now} }

// Answer writes down one card answered once, and returns the line as it was
// written: it carries the identifier that taking this answer back would name.
func (u Record) Answer(
	ctx context.Context, run *LogWriter, on review.CardFaceID, r review.Rating, took time.Duration,
) (review.Answer, error) {
	if !r.Valid() {
		return review.Answer{}, fmt.Errorf("%w: %d", ErrNoRating, r)
	}
	if on.Card == "" || on.Face == "" {
		return review.Answer{}, errors.New("an answer says which card, and through which face")
	}

	if u.now == nil {
		return review.Answer{}, errNoClock
	}
	at := u.now()
	id := ulid.New(at)
	a := review.Answer{ID: id, CardFace: on, At: at, Rating: r, Took: took}
	if err := run.Append(ctx, a); err != nil {
		return review.Answer{}, err
	}
	return a, nil
}

// TakeBack writes down that an answer was taken back. A person hits the wrong
// key, and the record of that is what says so.
func (u Record) TakeBack(ctx context.Context, run *LogWriter, id string) (review.Answer, error) {
	if id == "" {
		return review.Answer{}, ErrNoAnswer
	}
	if u.now == nil {
		return review.Answer{}, errNoClock
	}
	at := u.now()
	own := ulid.New(at)
	a := review.Answer{ID: own, At: at, Undoes: id}
	if err := run.Append(ctx, a); err != nil {
		return review.Answer{}, err
	}
	return a, nil
}
