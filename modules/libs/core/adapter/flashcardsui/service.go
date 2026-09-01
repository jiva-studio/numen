package flashcardsui

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// atOnce is how many vaults are counted alongside each other.
//
// Counting one reads every deck it holds and replays its whole answer log, so
// each of them holds a vault's worth of cards while it runs.
const atOnce = 4

// Owing counts every vault the installation knows.
//
// The vaults go first, by name and by where they are, and each count follows as
// it is worked out. A vault that could not be counted arrives with nothing
// counted and the reason on it. One vault the editor has never read is not a
// reason to refuse a person the others.
func (a *API) Owing(
	ctx context.Context, _ *connect.Request[v1.OwingRequest],
	out *connect.ServerStream[v1.OwingResponse],
) error {
	all, err := a.Registry.All()
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}

	listed := make([]*v1.VaultOwing, 0, len(all))
	for _, v := range all {
		listed = append(listed, &v1.VaultOwing{VaultId: v.ID, Name: v.Name, Path: v.Path})
	}
	// The day these counts stand in, which is the day a goal is weighed against.
	if err := out.Send(&v1.OwingResponse{Day: a.Day.Names(a.now()), Vaults: listed}); err != nil {
		return err
	}

	// A count left running is a vault read for a window that has gone, so the
	// counters are ended with this call whichever way it ends.
	ctx, stop := context.WithCancel(ctx)
	defer stop()

	for one := range a.counting(ctx, a.wanted(all)) {
		if err := out.Send(&v1.OwingResponse{Counted: one}); err != nil {
			return err
		}
	}
	return nil
}

// wanted is the order the vaults are counted in: the one opened last, then the
// rest as the registry holds them. A person coming back to this window is most
// often coming back to the vault they were last in.
func (a *API) wanted(all []domain.Vault) []domain.Vault {
	last, held, err := a.Registry.Last()
	if err != nil || !held {
		return all
	}
	at := slices.IndexFunc(all, func(v domain.Vault) bool { return v.ID == last.ID })
	if at <= 0 {
		return all
	}
	order := make([]domain.Vault, 0, len(all))
	order = append(order, all[at])
	order = append(order, all[:at]...)
	return append(order, all[at+1:]...)
}

// counting works the vaults out, a few at a time, and hands each over as it
// comes. A vault that takes a minute holds up nothing but the ones behind it in
// the queue.
func (a *API) counting(ctx context.Context, all []domain.Vault) <-chan *v1.VaultOwing {
	counted := make(chan *v1.VaultOwing)
	go func() {
		defer close(counted)
		var running sync.WaitGroup
		room := make(chan struct{}, atOnce)
		for _, v := range all {
			select {
			case room <- struct{}{}:
			case <-ctx.Done():
				running.Wait()
				return
			}
			running.Add(1)
			go func() {
				defer running.Done()
				defer func() { <-room }()
				select {
				case counted <- a.counted(ctx, v):
				case <-ctx.Done():
				}
			}()
		}
		running.Wait()
	}()
	return counted
}

func (a *API) counted(ctx context.Context, v domain.Vault) *v1.VaultOwing {
	one := &v1.VaultOwing{VaultId: v.ID, Name: v.Name, Path: v.Path}
	owing, err := a.Owed.Execute(ctx, v)
	if err != nil {
		one.Unread = err.Error()
		return one
	}
	one.Faces = int32(owing.Faces)
	one.Due = int32(owing.Due)
	one.New = int32(owing.New)
	for _, deck := range owing.Decks {
		one.Decks = append(one.Decks, &v1.DeckOwing{
			Deck:     deck.Deck,
			Faces:    int32(deck.Faces),
			Due:      int32(deck.Due),
			New:      int32(deck.New),
			Answered: int32(deck.Answered),
		})
	}
	for _, preset := range owing.Presets {
		one.Presets = append(one.Presets, &v1.PresetOwing{
			Preset:        preset.Preset,
			Title:         a.titled(ctx, v, preset.Preset),
			Decks:         int32(preset.Decks),
			Cards:         int32(preset.Cards),
			OwedDue:       int32(preset.Due),
			OwedNew:       int32(preset.New),
			Answered:      int32(preset.Answered),
			TookMs:        preset.Took.Milliseconds(),
			New:           int32(preset.Budget.New),
			Reviews:       int32(preset.Budget.Reviews),
			Minutes:       preset.Budget.Minutes,
			ClosesNew:     string(preset.Closes.New),
			ClosesReviews: string(preset.Closes.Reviews),
			ClosesMinutes: string(preset.Closes.Minutes),
			ClosesBacklog: string(preset.Closes.Backlog),
			StopsOn:       wire.StoppedOf(preset.Stops),
		})
	}
	return one
}

// Start opens a sitting and hands over what to ask, in order.
//
// A sitting is opened over one deck, over one preset, or over the whole vault.
// A refused sitting opens no run: the file a vault's answers go to is written
// when there is something to answer.
func (a *API) Start(
	ctx context.Context, r *connect.Request[v1.StartRequest],
) (*connect.Response[v1.StartResponse], error) {
	v, err := a.Vault(r.Msg.GetVaultId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	over := flashcards.Over{
		Deck:     r.Msg.GetDeck(),
		Preset:   r.Msg.GetPreset(),
		ByPreset: r.Msg.Preset != nil,
	}
	sitting, err := a.Session.Execute(ctx, v, over)
	if err != nil {
		switch {
		case errors.Is(err, flashcards.ErrBothNamed):
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		case errors.Is(err, flashcards.ErrUnread),
			errors.Is(err, flashcards.ErrSchedulesNothing):
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	run, err := a.opened(ctx, v)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := &v1.StartResponse{
		Run:       run.Name(),
		Asked:     make([]*v1.Asked, 0, len(sitting.Asked)),
		Unwritten: sitting.Unwritten,
		Skipped:   int32(sitting.Skipped),
	}
	for _, one := range sitting.Asked {
		out.Asked = append(out.Asked, askedOf(one))
	}
	return connect.NewResponse(out), nil
}

func askedOf(one flashcards.Asked) *v1.Asked {
	front, back := one.Lay()
	return &v1.Asked{
		Deck:    one.Deck,
		Section: one.Section,
		Card:    one.CardFace.Card,
		Face:    one.CardFace.Face,
		Heading: one.Heading,
		Front:   front,
		Back:    back,
		Seen:    one.Schedule.Seen(),
		Due:     stamp(one.Schedule.Due),
		Ahead:   ahead(one.Ahead),
	}
}

// ahead is where each of the four would leave the card, in seconds.
func ahead(said map[history.Rating]time.Duration) *v1.Ahead {
	if said == nil {
		return nil
	}
	return &v1.Ahead{
		Again: int64(said[history.Again].Seconds()),
		Hard:  int64(said[history.Hard].Seconds()),
		Good:  int64(said[history.Good].Seconds()),
		Easy:  int64(said[history.Easy].Seconds()),
	}
}

// Answer writes down how a card came back.
func (a *API) Answer(
	ctx context.Context, r *connect.Request[v1.AnswerRequest],
) (*connect.Response[v1.AnswerResponse], error) {
	v, err := a.Vault(r.Msg.GetVaultId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	run, err := a.running(v.ID, r.Msg.GetRun())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	on := history.CardFace{Card: r.Msg.GetCard(), Face: r.Msg.GetFace()}
	record := flashcards.Record{Run: run, Now: a.Now}
	given, err := record.Answer(ctx, on, rating(r.Msg.GetRating()),
		time.Duration(r.Msg.GetTookMs())*time.Millisecond)
	if err != nil {
		if errors.Is(err, flashcards.ErrNoRating) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.AnswerResponse{Answer: given.ID}), nil
}

// rating is the four a person may say. Anything else is refused by the use case,
// which is where the rule is.
func rating(r v1.Rating) history.Rating {
	switch r {
	case v1.Rating_RATING_AGAIN:
		return history.Again
	case v1.Rating_RATING_HARD:
		return history.Hard
	case v1.Rating_RATING_GOOD:
		return history.Good
	case v1.Rating_RATING_EASY:
		return history.Easy
	}
	return 0
}

// TakeBack writes down that an answer was taken back.
func (a *API) TakeBack(
	ctx context.Context, r *connect.Request[v1.TakeBackRequest],
) (*connect.Response[v1.TakeBackResponse], error) {
	v, err := a.Vault(r.Msg.GetVaultId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	run, err := a.running(v.ID, r.Msg.GetRun())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	record := flashcards.Record{Run: run, Now: a.Now}
	if _, err := record.TakeBack(ctx, r.Msg.GetAnswer()); err != nil {
		// An answer with no identifier is the caller's mistake; a line that
		// could not be written is not.
		if r.Msg.GetAnswer() == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.TakeBackResponse{}), nil
}
