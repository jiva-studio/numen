package flashcards

import (
	"context"
	"errors"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// ErrBothNamed is a session named a deck and a preset at once. Which cards were
// meant is a question, and it is put back to the caller.
var ErrBothNamed = errors.New("a session is opened over one deck or over one preset")

// ErrSchedulesNothing is a preset with nothing to ask in the day being sat. The
// message says which of the reasons it is, in the person's own words.
var ErrSchedulesNothing = errors.New("this preset schedules nothing today")

// Scope is what a session is opened over: every deck the vault holds, one of
// its decks, or one of its presets.
//
// The whole vault is the ordinary way to sit down to this: a person owes what
// they owe, and which file a card is written in is not something they think
// about.
type Scope struct {
	// Deck is the path of one deck.
	Deck string
	// Preset is the note one preset stands in. The preset scheduling the decks
	// naming none stands in no note, so Named says a preset was named at all.
	Preset string
	Named  bool
}

// OverDeck is a session over one deck.
func OverDeck(path string) Scope { return Scope{Deck: path} }

// ByPreset is a session over the cards of every deck pointing at one preset,
// held to that preset's budget.
func ByPreset(preset string) Scope { return Scope{Preset: preset, Named: true} }

// QueuedCardFace is one card face as the session queues it: where it stands,
// how it is laid out, and where the answers so far have left it.
type QueuedCardFace struct {
	CardFace
	Schedule review.Schedule
	// Ahead is how long each of the four answers would leave this card, from
	// the moment it is asked. A person choosing between them is choosing
	// between these, so they are worked out with the card and not after it.
	Ahead map[review.Rating]time.Duration
}

// SessionResult is what a person sits down to: the cards to ask, and what could
// not be acted on in getting them.
type SessionResult struct {
	// Queue is what to ask, in the order to ask it.
	Queue []QueuedCardFace
	// Unwritten are the decks holding a card with no mark that could not be
	// given one. Their cards are not in the queue and are asked for at the next
	// session.
	Unwritten []string
	// Skipped is how many lines of the vault's answers could not be read: a run
	// that stopped partway, or a line of a version this build does not know.
	Skipped int
}

// Session is what a person is asked, in the order they are asked it.
//
// A card owed and answered before comes first, the one waiting longest at the
// front, because a card left late is the one closest to being forgotten. Cards
// nobody has answered come after them, in the order they stand in their decks:
// a person wrote them in an order, and it is as good an order as any.
//
// Each deck is held to the budget its preset keeps today, and what was already
// answered today is off that budget.
type Session struct {
	// Marks gives a mark to the cards of this vault that carry none, so that
	// what is asked can be answered. It is the one write flashcards makes, and it
	// is made when a person sits down to a vault.
	Marks     MarkCards
	CardFaces ListCardFaces
	Schedules Schedules
	// Presets says which preset each deck is scheduled by. A build holding no
	// links schedules every deck by the defaults.
	Presets Presets
	Day     review.Day
	Now     port.Clock
}

// NewSession is what a person sits down to their cards through: what gives the
// unmarked cards their marks, what stands in the vault, where the answers have
// left each card face, which preset each deck is scheduled by, where one day of
// review gives way to the next, and what time it is.
func NewSession(
	marks MarkCards, faces ListCardFaces, schedules Schedules, presets Presets,
	day review.Day, now port.Clock,
) Session {
	return Session{
		Marks: marks, CardFaces: faces, Schedules: schedules, Presets: presets,
		Day: day, Now: now,
	}
}

// Execute is what to ask, in order.
//
// Scope is the deck or the preset the session is opened over. Naming both is
// ErrBothNamed, and a preset with nothing to ask today is ErrSchedulesNothing
// with the reason.
func (u Session) Execute(
	ctx context.Context, v domain.Vault, over Scope,
) (SessionResult, error) {
	if over.Named && over.Deck != "" {
		return SessionResult{}, ErrBothNamed
	}
	marked, err := u.Marks.Execute(ctx, v)
	if err != nil {
		return SessionResult{}, err
	}
	faces, err := u.CardFaces.Execute(ctx, v)
	if err != nil {
		return SessionResult{}, err
	}

	// The log is read once here and the schedules worked out from it, so that
	// what a person is told about lines that could not be read is the reading
	// their own cards were laid out from.
	held, err := Log{Stores: u.Schedules.Logs}.Read(ctx, v)
	if err != nil {
		return SessionResult{}, err
	}
	// One reading of this vault's presets answers both the schedulers the cards
	// are worked out by and the budgets they are held to.
	reading := u.Presets.Reading()
	asks, err := u.Schedules.under(ctx, v, reading, faces)
	if err != nil {
		return SessionResult{}, err
	}
	schedules := u.Schedules.getSchedulesCached(ctx, v, held, asks)

	now := u.Now()
	day, err := getBudgets(
		ctx, v, reading, u.Day, faces, schedules, held,
		u.Schedules.By, u.Schedules.at, now,
	)
	if err != nil {
		return SessionResult{}, err
	}
	holds := day.asks(faces, schedules, u.Day, now, over)
	if over.Named && len(holds.seen)+len(holds.fresh) == 0 {
		return SessionResult{}, day.refuses(over.Preset)
	}

	// How loaded each day of review already is, which is what a card put on one
	// of them is weighed against.
	on := review.NewDueByDay(u.Day)
	for _, s := range schedules {
		on.Holds(s.Due)
	}

	out := SessionResult{Unwritten: marked.Unwritten, Skipped: held.Skipped}
	out.Queue = make([]QueuedCardFace, 0, len(holds.seen)+len(holds.fresh))
	for _, one := range holds.seen {
		s := schedules[one.ID]
		out.Queue = append(out.Queue, QueuedCardFace{
			CardFace: one, Schedule: s, Ahead: ahead(asks.under, on, one.ID, s, now),
		})
	}
	for _, one := range holds.fresh {
		out.Queue = append(out.Queue, QueuedCardFace{
			CardFace: one,
			Ahead:    ahead(asks.under, on, one.ID, review.Schedule{}, now),
		})
	}
	return out, nil
}

// ahead is how long each of the four would leave a card standing where this
// schedule leaves it. The card is left where it was, and so is the table of how
// loaded each day is: only the answer a person gives lands anywhere.
//
// The scheduler is the one this card face is scheduled by and the placement is
// its own preset's, so the window under each button is the day the card will
// come back on.
func ahead(
	under review.Assignment, on *review.DueByDay, face review.CardFaceID,
	s review.Schedule, now time.Time,
) map[review.Rating]time.Duration {
	if under == nil {
		return nil
	}
	one := under(face)
	if one.By == nil {
		return nil
	}
	out := make(map[review.Rating]time.Duration, 4)
	for _, r := range []review.Rating{review.Again, review.Hard, review.Good, review.Easy} {
		due := one.By.Next(s, now, r).Due
		out[r] = one.Preset.Lands(on, now, due).Sub(now)
	}
	return out
}
