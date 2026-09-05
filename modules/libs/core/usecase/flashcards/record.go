package flashcards

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	"github.com/jiva-studio/numen/modules/libs/core/internal/ulid"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// ErrNoRating is what an answer outside the four gets. Nothing here guesses
// what a person meant to say about their own recall.
var ErrNoRating = errors.New("not one of the four ratings")

// Record writes down what a person answered.
//
// A line is appended and nothing is ever rewritten, so what was recorded stands
// whatever happens next — including a person taking an answer back, which is a
// line of its own naming the one it takes back.
type Record struct {
	Run *LogWriter
	Now port.Clock
}

// Answer writes down one card answered once, and hands back the line as it was
// written: it carries the identifier that taking this answer back would name.
func (u Record) Answer(
	ctx context.Context, on review.CardFaceID, r review.Rating, took time.Duration,
) (review.Answer, error) {
	if !r.Valid() {
		return review.Answer{}, fmt.Errorf("%w: %d", ErrNoRating, r)
	}
	if on.Card == "" || on.Face == "" {
		return review.Answer{}, errors.New("an answer says which card, and through which face")
	}

	at := u.Now()
	id, err := ulid.New(at)
	if err != nil {
		return review.Answer{}, err
	}
	a := review.Answer{ID: id, CardFace: on, At: at, Rating: r, Took: took}
	if err := u.Run.Append(ctx, a); err != nil {
		return review.Answer{}, err
	}
	return a, nil
}

// TakeBack writes down that an answer was taken back. A person hits the wrong
// key, and the record of that is what says so.
func (u Record) TakeBack(ctx context.Context, id string) (review.Answer, error) {
	if id == "" {
		return review.Answer{}, errors.New("an answer taken back names the one it takes back")
	}
	at := u.Now()
	own, err := ulid.New(at)
	if err != nil {
		return review.Answer{}, err
	}
	a := review.Answer{ID: own, At: at, Undoes: id}
	if err := u.Run.Append(ctx, a); err != nil {
		return review.Answer{}, err
	}
	return a, nil
}
