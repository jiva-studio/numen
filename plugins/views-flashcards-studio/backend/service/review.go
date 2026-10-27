package service

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	flashcardsv1 "github.com/jiva-studio/numen/plugins/views-flashcards-studio/gen/numen/flashcards/v1"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/flashcards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/service/presetwire"
)

// ErrNoRun is an answer named against a run this backend did not open. A run is one session at one
// window, and an answer belongs to the session it was given in.
var ErrNoRun = errors.New("no run of that name is open")

// reviewHandler answers the window a person runs their cards in.
//
// The runs it holds are the sessions open here, one to a vault. A run is a file in a vault that only
// this backend appends to, and it is release when another session opens on that vault: what was
// written stands, and the next session opens a file of its own.
type reviewHandler struct {
	*Backend
}

// sessions is the run open on each vault, by the vault's identity. One session to a vault: opening
// another lets go of the one before it, and the file that one wrote is never appended to again.
type sessions struct {
	mu  sync.Mutex
	run map[domain.VaultID]*flashcards.LogWriter
}

func (s *sessions) store(v domain.Vault, run *flashcards.LogWriter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.run == nil {
		s.run = make(map[domain.VaultID]*flashcards.LogWriter)
	}
	s.run[v.ID] = run
}

// getRun is the run this backend has open on a vault, and only under the name that run writes.
//
// The vault is part of what is asked for, because an answer is written to the vault its run was
// opened on: a run named against another vault would put a person's answer in a history it does not
// belong to, and an answer written is not written again. The name is asked for as well, so a page
// holding the name of a session that is over cannot go on writing to it.
func (s *sessions) getRun(vault domain.VaultID, name string) (*flashcards.LogWriter, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, is := s.run[vault]
	if !is || run.GetName() != name {
		return nil, fmt.Errorf("%w: %s", ErrNoRun, name)
	}
	return run, nil
}

// person's session is on.
func (h *reviewHandler) openRun(ctx context.Context, v domain.Vault) (*flashcards.LogWriter, error) {
	run, err := h.scenarios().flash.Log.Open(ctx, v, h.now())
	if err != nil {
		return nil, err
	}
	h.runs.store(v, run)
	return run, nil
}

// formatTime is how an instant reaches the page: the one shape everything here writes, so the page
// parses one.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(review.TimestampLayout)
}

// atOnce is how many vaults are counted alongside each other.
//
// Counting one reads every deck it holds and replays its whole answer log, so each of them holds a
// vault's worth of cards while it runs.
const atOnce = 4

// WatchCardsDue counts every vault the installation knows.
//
// The vaults go first, by name and by where they are, and each count follows as
// it is worked out. A vault that could not be counted arrives with nothing
// counted and the reason on it. One vault the editor has never read is not a
// reason to refuse a person the others.
func (a *reviewHandler) WatchCardsDue(
	ctx context.Context, _ *connect.Request[flashcardsv1.WatchCardsDueRequest],
	out *connect.ServerStream[flashcardsv1.WatchCardsDueResponse],
) error {
	recognized, err := a.vault.GetVaults(ctx)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}

	items := make([]*flashcardsv1.VaultCardsDue, 0, len(recognized))
	for _, one := range recognized {
		items = append(items, &flashcardsv1.VaultCardsDue{Id: string(one.ID), Name: one.Name, Path: one.Path})
	}
	// The day these counts stand in, which is the day a goal is weighed against.
	createResponse := func() *flashcardsv1.WatchCardsDueResponse {
		return &flashcardsv1.WatchCardsDueResponse{Day: a.scenarios().day.GetName(a.now()), Vaults: items}
	}
	if err := out.Send(createResponse()); err != nil {
		return err
	}

	// A count left running is a vault read for a window that has gone, so the
	// counters are ended with this call whichever way it ends.
	ctx, stop := context.WithCancel(ctx)
	defer stop()

	// Counting four vaults reads four vaults, and nothing else tells this
	// handler its client has gone: the request context belongs to the process
	// and is not cancelled when a page is reloaded away from under the stream.
	// A write that fails is the one report there is, so the list goes again
	// while nothing has been counted, and a person reloading twice does not
	// leave two sets of counters reading the vault.
	repeat := time.NewTicker(again)
	defer repeat.Stop()

	counted := a.countVaults(ctx, recognized)
	for counted != nil {
		select {
		case one, open := <-counted:
			if !open {
				counted = nil
				continue
			}
			if err := out.Send(&flashcardsv1.WatchCardsDueResponse{Counted: one}); err != nil {
				return err
			}
			repeat.Reset(again)
		case <-repeat.C:
			if err := out.Send(createResponse()); err != nil {
				return err
			}
		}
	}
	return nil
}

// countVaults works the vaults out, a few at a time, and hands each over as it
// comes. A vault that takes a minute holds up nothing but the ones behind it in
// the queue.
func (a *reviewHandler) countVaults(ctx context.Context, all []domain.Vault) <-chan *flashcardsv1.VaultCardsDue {
	counted := make(chan *flashcardsv1.VaultCardsDue)
	go func() {
		defer close(counted)
		var running sync.WaitGroup
		room := make(chan struct{}, atOnce)
		for _, one := range all {
			v := one
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
				case counted <- a.countVault(ctx, v):
				case <-ctx.Done():
				}
			}()
		}
		running.Wait()
	}()
	return counted
}

func (a *reviewHandler) countVault(ctx context.Context, v domain.Vault) *flashcardsv1.VaultCardsDue {
	one := &flashcardsv1.VaultCardsDue{Id: string(v.ID), Name: v.Name, Path: v.Path}

	// The vault is brought up to date before it is counted. Nothing is counted from a walk half done,
	// and the numbers arrive with the count that the finished walk wakes.
	state, err := a.vault.With(v.ID).EnsureIndexed(ctx)
	if err != nil {
		one.Unread = err.Error()
		return one
	}
	switch state.Status {
	case host.Indexing:
		one.IsIndexing = true
		return one
	case host.IndexFailed:
		one.Unread = state.Reason
		return one
	case host.Indexed:
	}

	owing, err := a.scenarios().flash.CardsDue.Execute(ctx, v)
	if err != nil {
		one.Unread = err.Error()
		return one
	}
	one.ReviewCards = int32(owing.ReviewCards)
	one.Due = int32(owing.Due)
	one.New = int32(owing.New)
	for _, deck := range owing.Decks {
		one.Decks = append(one.Decks, &flashcardsv1.DeckCardsDue{
			Deck:        deck.Deck,
			ReviewCards: int32(deck.ReviewCards),
			Due:         int32(deck.Due),
			New:         int32(deck.New),
			Answered:    int32(deck.Answered),
			Learned:     int32(deck.Learned),
			Unbegun:     int32(deck.Unbegun),
		})
	}
	for _, preset := range owing.Presets {
		one.Presets = append(one.Presets, &flashcardsv1.PresetCardsDue{
			Preset:          preset.Preset,
			Title:           a.getTitle(ctx, v, preset.Preset),
			Decks:           int32(preset.Decks),
			Cards:           int32(preset.Cards),
			OwedDue:         int32(preset.Due),
			OwedNew:         int32(preset.New),
			Answered:        int32(preset.Answered),
			AnsweredNew:     int32(preset.AnsweredNew),
			AnsweredReviews: int32(preset.AnsweredReviews),
			TookMs:          preset.Took.Milliseconds(),
			New:             int32(preset.Budget.New),
			Reviews:         int32(preset.Budget.Reviews),
			Minutes:         preset.Budget.Minutes,
			ClosesNew:       string(preset.Limits.New),
			ClosesReviews:   string(preset.Limits.Reviews),
			ClosesMinutes:   string(preset.Limits.Minutes),
			ClosesBacklog:   string(preset.Limits.Backlog),
			StopsOn:         presetwire.ConvertStopReasonToWire(preset.Stops),
		})
	}
	return one
}

// StartSession opens a session and hands over what to ask, in order.
//
// A session is opened over one deck, over one preset, or over the whole vault.
// when there is something to answer.
func (a *reviewHandler) StartSession(
	ctx context.Context, r *connect.Request[flashcardsv1.StartSessionRequest],
) (*connect.Response[flashcardsv1.StartSessionResponse], error) {
	v, err := a.getVault(ctx, r.Msg.GetVault())
	if err != nil {
		return nil, err
	}
	cardScope := flashcards.Scope{
		Deck:    r.Msg.GetDeck(),
		Preset:  r.Msg.GetPreset(),
		IsNamed: r.Msg.Preset != nil,
	}
	session, err := a.scenarios().flash.Session.Execute(ctx, v, cardScope)
	if err != nil {
		switch {
		case errors.Is(err, flashcards.ErrBothNamed):
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		case errors.Is(err, flashcards.ErrNotCarried),
			errors.Is(err, flashcards.ErrSchedulesNothing):
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	run, err := a.openRun(ctx, v)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := &flashcardsv1.StartSessionResponse{
		Run:       run.GetName(),
		Cards:     make([]*flashcardsv1.SessionCard, 0, len(session.Queue)),
		Unwritten: session.Unwritten,
		Skipped:   int32(session.Skipped),
	}
	for _, one := range session.Queue {
		out.Cards = append(out.Cards, newWireSessionCard(one))
	}
	return connect.NewResponse(out), nil
}

func newWireSessionCard(one flashcards.QueuedReviewCard) *flashcardsv1.SessionCard {
	front, back := one.Lay()
	return &flashcardsv1.SessionCard{
		Deck:      one.Deck,
		Section:   one.Section,
		CardId:    one.ID.Card,
		Template:  one.ID.Template,
		Heading:   one.Heading,
		Front:     front,
		Back:      back,
		IsNew:     one.Schedule.IsNew(),
		Due:       formatTime(one.Schedule.Due),
		Intervals: newWireIntervals(one.Ahead),
	}
}

// newWireIntervals is where each of the four would leave the card, in seconds.
func newWireIntervals(ahead map[review.Rating]time.Duration) *flashcardsv1.Intervals {
	if ahead == nil {
		return nil
	}
	return &flashcardsv1.Intervals{
		Again: int64(ahead[review.Again].Seconds()),
		Hard:  int64(ahead[review.Hard].Seconds()),
		Good:  int64(ahead[review.Good].Seconds()),
		Easy:  int64(ahead[review.Easy].Seconds()),
	}
}

// AnswerCard writes down how a card came back.
func (a *reviewHandler) AnswerCard(
	ctx context.Context, r *connect.Request[flashcardsv1.AnswerCardRequest],
) (*connect.Response[flashcardsv1.AnswerCardResponse], error) {
	v, err := a.getVault(ctx, r.Msg.GetVault())
	if err != nil {
		return nil, err
	}
	run, err := a.runs.getRun(v.ID, r.Msg.GetRun())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	on := review.ReviewCardID{Card: r.Msg.GetCardId(), Template: r.Msg.GetTemplate()}
	provided, err := a.scenarios().flash.Record.Answer(ctx, run, on, parseRating(r.Msg.GetRating()),
		time.Duration(r.Msg.GetTookMs())*time.Millisecond)
	if err != nil {
		if errors.Is(err, flashcards.ErrNoRating) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&flashcardsv1.AnswerCardResponse{Answer: provided.ID}), nil
}

// case, which is where the rule is.
func parseRating(r flashcardsv1.Rating) review.Rating {
	switch r {
	case flashcardsv1.Rating_RATING_AGAIN:
		return review.Again
	case flashcardsv1.Rating_RATING_HARD:
		return review.Hard
	case flashcardsv1.Rating_RATING_GOOD:
		return review.Good
	case flashcardsv1.Rating_RATING_EASY:
		return review.Easy
	default:
		return 0
	}
}

// TakeBackAnswer writes down that an answer was taken back.
func (a *reviewHandler) TakeBackAnswer(
	ctx context.Context, r *connect.Request[flashcardsv1.TakeBackAnswerRequest],
) (*connect.Response[flashcardsv1.TakeBackAnswerResponse], error) {
	v, err := a.getVault(ctx, r.Msg.GetVault())
	if err != nil {
		return nil, err
	}
	run, err := a.runs.getRun(v.ID, r.Msg.GetRun())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if _, err := a.scenarios().flash.Record.TakeBack(ctx, run, r.Msg.GetAnswer()); err != nil {
		if errors.Is(err, flashcards.ErrNoAnswer) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&flashcardsv1.TakeBackAnswerResponse{}), nil
}

// GetVaultDeckPreset is the preset a deck of the named vault is scheduled by. A
// deck naming none is answered with the defaults under no path.
func (a *reviewHandler) GetVaultDeckPreset(
	ctx context.Context, r *connect.Request[flashcardsv1.GetVaultDeckPresetRequest],
) (*connect.Response[flashcardsv1.GetVaultDeckPresetResponse], error) {
	v, err := a.getVault(ctx, r.Msg.GetVault())
	if err != nil {
		return nil, err
	}
	found, err := a.scenarios().flash.Presets.GetForDeck(ctx, v, r.Msg.GetDeck())
	if err != nil {
		return nil, connect.NewError(getCode(err), err)
	}

	out := &flashcardsv1.GetVaultDeckPresetResponse{}
	if reason, ok := convertOutcomeToErrorCode(found.Outcome); ok {
		out.Error = &reason
		return connect.NewResponse(out), nil
	}
	out.Preset = presetwire.ConvertPresetToWire(found, a.getTitle(ctx, v, found.Path))
	return connect.NewResponse(out), nil
}

// ListReviewDays is how much of a vault was answered on each day it was
// reviewed.
//
// The days come back in order, oldest first, because what draws them draws them
// along a line of time. A caller naming a stretch is answered that stretch; the
// streak and the count are over the whole of the vault either way.
func (a *reviewHandler) ListReviewDays(
	ctx context.Context, r *connect.Request[flashcardsv1.ListReviewDaysRequest],
) (*connect.Response[flashcardsv1.ListReviewDaysResponse], error) {
	v, err := a.getVault(ctx, r.Msg.GetVault())
	if err != nil {
		return nil, err
	}
	response, err := a.scenarios().flash.GetReviewDays.Execute(ctx, v, r.Msg.GetFrom(), r.Msg.GetTo())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	days := make([]*flashcardsv1.ReviewDay, 0, len(response.Days))
	for day, one := range response.Days {
		came := response.Retained[day]
		days = append(days, &flashcardsv1.ReviewDay{
			Day:      day,
			Answered: int32(one.Answered),
			Again:    int32(one.Again),
			Hard:     int32(one.Hard),
			Good:     int32(one.Good),
			Easy:     int32(one.Easy),
			Asked:    int32(came.Asked),
			Recalled: int32(came.Recalled),
		})
	}
	due := make([]*flashcardsv1.ReviewDay, 0, len(response.Due))
	for day, falls := range response.Due {
		due = append(due, &flashcardsv1.ReviewDay{Day: day, Answered: int32(falls)})
	}

	return connect.NewResponse(&flashcardsv1.ListReviewDaysResponse{
		Days:     sortReviewDays(days),
		Due:      sortReviewDays(due),
		Streak:   int32(response.Streak),
		Answered: int32(response.Answered),
	}), nil
}

// sortReviewDays puts the days oldest first.
func sortReviewDays(days []*flashcardsv1.ReviewDay) []*flashcardsv1.ReviewDay {
	slices.SortFunc(days, func(one, other *flashcardsv1.ReviewDay) int {
		return cmp.Compare(one.GetDay(), other.GetDay())
	})
	return days
}

// GetDeckNeighbourhood hands over what the deck being sat to is joined to.
//
// Nothing here writes and no run is named: a person may read around a deck
// before they have started on it, and reading is not part of a session.
func (a *reviewHandler) GetDeckNeighbourhood(
	ctx context.Context, r *connect.Request[flashcardsv1.GetDeckNeighbourhoodRequest],
) (*connect.Response[flashcardsv1.GetDeckNeighbourhoodResponse], error) {
	v, err := a.getVault(ctx, r.Msg.GetVault())
	if err != nil {
		return nil, err
	}
	joined, err := a.scenarios().neighbourhood.Execute(ctx, v, r.Msg.GetDeck())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := &flashcardsv1.GetDeckNeighbourhoodResponse{
		Notes:  make([]*flashcardsv1.DeckNeighbour, 0, len(joined.Notes)),
		Unread: int32(joined.Unread),
	}
	for _, one := range joined.Notes {
		next := &flashcardsv1.DeckNeighbour{
			Written:     one.Written,
			Path:        one.Path,
			Title:       one.Title,
			Body:        one.Body,
			Label:       one.Label,
			IsPointedAt: !one.IsBacklink,
			IsAmbiguous: one.IsAmbiguous,
		}
		if reason, ok := convertOutcomeToErrorCode(one.Outcome); ok {
			next.Error = &reason
		}
		out.Notes = append(out.Notes, next)
	}
	return connect.NewResponse(out), nil
}

// again is how often a stream that has seen nothing move says so, so that a page can tell a
// stream that is open from one that is not.
const again = time.Second

// WatchReloads says something moved, for as long as the caller listens.
//
// What moved is not carried: what this window shows is counts and the cards behind them, and they are
// asked for again whatever changed.
func (a *reviewHandler) WatchReloads(
	ctx context.Context,
	_ *connect.Request[flashcardsv1.WatchReloadsRequest],
	out *connect.ServerStream[flashcardsv1.WatchReloadsResponse],
) error {
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	moved, err := a.vault.WatchMoves(ctx)
	if err != nil {
		return connect.NewError(connect.CodeUnavailable, err)
	}

	// Named as listening before anything has moved. A stream that says nothing until a file changes
	// cannot be told from one that never opened.
	if err := out.Send(&flashcardsv1.WatchReloadsResponse{}); err != nil {
		return err
	}

	repeat := time.NewTicker(again)
	defer repeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-repeat.C:
			// Nothing moved, which is what the opening message says too. A page that has gone fails the
			// write.
			if err := out.Send(&flashcardsv1.WatchReloadsResponse{}); err != nil {
				return err
			}
		case _, open := <-moved:
			if !open {
				return nil
			}
			a.scenarios().flash.ReviewCards.InvalidateAll()
			if err := out.Send(&flashcardsv1.WatchReloadsResponse{Reload: true}); err != nil {
				return err
			}
		}
	}
}
