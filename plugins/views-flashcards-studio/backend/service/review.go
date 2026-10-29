package service

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/flashcards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/adapter/clock"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/service/presetwire"
)

// ErrNoRun is an answer named against a run this backend did not open. A run is one session at one
// window, and an answer belongs to the session it was given in.
var ErrNoRun = errors.New("no run of that name is open")

// reviewHandler serves the review window.
//
// The runs it holds are the sessions open here, one per vault. A run is a file in a vault that only
// this backend appends to. It is released when another session opens on that vault: the written
// answers remain, and the next session opens a new file.
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

// getRun returns the run this backend has open on a vault, found only under the name that run writes.
//
// The vault is part of the lookup, because an answer is written to the vault its run was
// opened on: a run named against another vault would put an answer in the wrong history, and a
// written answer is not written again. The name is part of the lookup as well, so a page
// holding the name of a finished session cannot continue writing to it.
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

// formatTime formats an instant for the page: the one shape everything here writes, so the page
// parses a single format.
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

// WatchCardsDue counts every vault the installation contains.
//
// The vaults are sent first, with name and location, and each count follows as
// it is computed. A vault that cannot be counted is sent with no count and the
// error. A single unreadable vault does not block the counts of the others.
func (h *reviewHandler) WatchCardsDue(
	ctx context.Context, _ *api.WatchCardsDueRequest,
	send func(*api.WatchCardsDueResponse) error,
) error {
	recognized, err := h.vault.GetVaults(ctx)
	if err != nil {
		return newError(server.CodeInternal, err)
	}

	items := make([]*api.VaultCardsDue, 0, len(recognized))
	for _, one := range recognized {
		items = append(items, newVaultCardsDue(one))
	}
	// The day these counts stand in, which is the day a goal is weighed against.
	createResponse := func() *api.WatchCardsDueResponse {
		return &api.WatchCardsDueResponse{Vaults: items, Day: h.scenarios().day.GetName(h.now()), Counted: nil}
	}
	if err := send(createResponse()); err != nil {
		return err
	}

	// A count left running is a vault read for a window that has gone, so the
	// counters are ended with this call whichever way it ends.
	ctx, stop := context.WithCancel(ctx)
	defer stop()

	// Counting four vaults reads four vaults, and nothing else signals this
	// handler that its client has gone: the request context belongs to the process
	// and is not cancelled when a page is reloaded during the stream.
	// A failed write is the only signal, so the list is sent again periodically
	// while nothing has been counted, and a repeated reload does not
	// leave two sets of counters reading the vault.
	repeat := clock.NewTicker(again)
	defer repeat.Stop()

	counted := h.countVaults(ctx, recognized)
	for counted != nil {
		select {
		case one, open := <-counted:
			if !open {
				counted = nil
				continue
			}
			if err := send(&api.WatchCardsDueResponse{Vaults: nil, Day: "", Counted: one}); err != nil {
				return err
			}
			repeat.Reset(again)
		case <-repeat.C:
			if err := send(createResponse()); err != nil {
				return err
			}
		}
	}
	return nil
}

// countVaults works the vaults out, a few at a time, and hands each over as it
// comes. A vault that takes a minute holds up nothing but the ones behind it in
// the queue.
func (h *reviewHandler) countVaults(ctx context.Context, all []domain.Vault) <-chan *api.VaultCardsDue {
	counted := make(chan *api.VaultCardsDue)
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
				case counted <- h.countVault(ctx, v):
				case <-ctx.Done():
				}
			}()
		}
		running.Wait()
	}()
	return counted
}

// newVaultCardsDue returns a vault entry that carries no count.
func newVaultCardsDue(v domain.Vault) *api.VaultCardsDue {
	return &api.VaultCardsDue{
		ID: string(v.ID), Name: v.Name, Path: v.Path,
		ReviewCards: 0, Due: 0, New: 0, Decks: nil, Presets: nil, Unread: "", IsIndexing: false,
	}
}

func (h *reviewHandler) countVault(ctx context.Context, v domain.Vault) *api.VaultCardsDue {
	one := newVaultCardsDue(v)

	// The vault is brought up to date before it is counted. Nothing is counted from a walk half done,
	// and the numbers arrive with the count that the finished walk wakes.
	state, err := h.vault.With(v.ID).EnsureIndexed(ctx)
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

	owing, err := h.scenarios().flash.CardsDue.Execute(ctx, v)
	if err != nil {
		one.Unread = err.Error()
		return one
	}
	one.ReviewCards = int32(owing.ReviewCards)
	one.Due = int32(owing.Due)
	one.New = int32(owing.New)
	for _, deck := range owing.Decks {
		one.Decks = append(one.Decks, &api.DeckCardsDue{
			Deck:        deck.Deck,
			ReviewCards: int32(deck.ReviewCards),
			Due:         int32(deck.Due),
			New:         int32(deck.New),
			Reviewed:    int32(deck.Answered),
			Learned:     int32(deck.Learned),
			Unbegun:     int32(deck.Unbegun),
		})
	}
	for _, preset := range owing.Presets {
		one.Presets = append(one.Presets, &api.PresetCardsDue{
			Preset:          preset.Preset,
			Title:           h.getTitle(ctx, v, preset.Preset),
			Decks:           int32(preset.Decks),
			Cards:           int32(preset.Cards),
			OwedDue:         int32(preset.Due),
			OwedNew:         int32(preset.New),
			Reviewed:        int32(preset.Answered),
			ReviewedNew:     int32(preset.ReviewedNew),
			ReviewedReviews: int32(preset.ReviewedReviews),
			DurationMs:      preset.Took.Milliseconds(),
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

// StartSession opens a session and returns the cards to ask, in order.
//
// A session is opened over one deck, over one preset, or over the whole vault.
// The cards are returned only when there is something to answer.
func (h *reviewHandler) StartSession(
	ctx context.Context, req *api.StartSessionRequest,
) (*api.StartSessionResponse, error) {
	v, err := h.getVault(ctx, req.Vault)
	if err != nil {
		return nil, err
	}
	cardScope := flashcards.Scope{Deck: req.Deck, Preset: "", IsNamed: req.Preset != nil}
	if req.Preset != nil {
		cardScope.Preset = *req.Preset
	}
	session, err := h.scenarios().flash.Session.Execute(ctx, v, cardScope)
	if err != nil {
		switch {
		case errors.Is(err, flashcards.ErrBothNamed):
			return nil, newError(server.CodeInvalidArgument, err)
		case errors.Is(err, flashcards.ErrNotCarried),
			errors.Is(err, flashcards.ErrSchedulesNothing):
			return nil, newError(server.CodeFailedPrecondition, err)
		}
		return nil, newError(server.CodeInternal, err)
	}
	run, err := h.openRun(ctx, v)
	if err != nil {
		return nil, newError(server.CodeInternal, err)
	}

	out := &api.StartSessionResponse{
		Run:       run.GetName(),
		Cards:     make([]*api.SessionCard, 0, len(session.Queue)),
		Unwritten: session.Unwritten,
		Skipped:   int32(session.Skipped),
	}
	for _, one := range session.Queue {
		out.Cards = append(out.Cards, newWireSessionCard(one))
	}
	return out, nil
}

func newWireSessionCard(one flashcards.QueuedReviewCard) *api.SessionCard {
	front, back := one.Lay()
	return &api.SessionCard{
		Deck:      one.Deck,
		Section:   one.Section,
		CardID:    one.ID.Card,
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
func newWireIntervals(ahead map[review.Rating]time.Duration) *api.Intervals {
	if ahead == nil {
		return nil
	}
	return &api.Intervals{
		Again: int64(ahead[review.Again].Seconds()),
		Hard:  int64(ahead[review.Hard].Seconds()),
		Good:  int64(ahead[review.Good].Seconds()),
		Easy:  int64(ahead[review.Easy].Seconds()),
	}
}

// RecordReview writes down how a card came back.
func (h *reviewHandler) RecordReview(
	ctx context.Context, req *api.RecordReviewRequest,
) (*api.RecordReviewResponse, error) {
	v, err := h.getVault(ctx, req.Vault)
	if err != nil {
		return nil, err
	}
	run, err := h.runs.getRun(v.ID, req.Run)
	if err != nil {
		return nil, newError(server.CodeNotFound, err)
	}

	on := review.CardID{Card: req.CardID, Template: req.Template}
	provided, err := h.scenarios().flash.Record.Answer(ctx, run, on, parseRating(req.Rating),
		time.Duration(req.DurationMs)*time.Millisecond)
	if err != nil {
		if errors.Is(err, flashcards.ErrNoRating) {
			return nil, newError(server.CodeInvalidArgument, err)
		}
		return nil, newError(server.CodeInternal, err)
	}

	return &api.RecordReviewResponse{Answer: provided.ID}, nil
}

// case, which is where the rule is.
func parseRating(r api.Rating) review.Rating {
	switch r {
	case api.RatingAgain:
		return review.Again
	case api.RatingHard:
		return review.Hard
	case api.RatingGood:
		return review.Good
	case api.RatingEasy:
		return review.Easy
	case api.RatingUnspecified:
		return 0
	default:
		return 0
	}
}

// UndoReview writes down that an answer was taken back.
func (h *reviewHandler) UndoReview(
	ctx context.Context, req *api.UndoReviewRequest,
) (*api.UndoReviewResponse, error) {
	v, err := h.getVault(ctx, req.Vault)
	if err != nil {
		return nil, err
	}
	run, err := h.runs.getRun(v.ID, req.Run)
	if err != nil {
		return nil, newError(server.CodeNotFound, err)
	}
	if _, err := h.scenarios().flash.Record.TakeBack(ctx, run, req.Answer); err != nil {
		if errors.Is(err, flashcards.ErrNoAnswer) {
			return nil, newError(server.CodeInvalidArgument, err)
		}
		return nil, newError(server.CodeInternal, err)
	}
	return &api.UndoReviewResponse{}, nil
}

// GetVaultDeckPreset is the preset a deck of the named vault is scheduled by. A
// deck naming none is answered with the defaults under no path.
func (h *reviewHandler) GetVaultDeckPreset(
	ctx context.Context, req *api.GetVaultDeckPresetRequest,
) (*api.GetVaultDeckPresetResponse, error) {
	v, err := h.getVault(ctx, req.Vault)
	if err != nil {
		return nil, err
	}
	found, err := h.scenarios().flash.Presets.GetForDeck(ctx, v, req.Deck)
	if err != nil {
		return nil, newError(getCode(err), err)
	}

	out := &api.GetVaultDeckPresetResponse{Preset: nil, Error: api.ErrorCodeUnspecified}
	if reason, ok := convertOutcomeToErrorCode(found.Outcome); ok {
		out.Error = reason
		return out, nil
	}
	out.Preset = presetwire.ConvertPresetToWire(found, h.getTitle(ctx, v, found.Path))
	return out, nil
}

// ListReviewDays is how much of a vault was answered on each day it was
// reviewed.
//
// The days come back in order, oldest first, because what draws them draws them
// along a line of time. A caller naming a stretch is answered that stretch; the
// streak and the count are over the whole of the vault either way.
func (h *reviewHandler) ListReviewDays(
	ctx context.Context, req *api.ListReviewDaysRequest,
) (*api.ListReviewDaysResponse, error) {
	v, err := h.getVault(ctx, req.Vault)
	if err != nil {
		return nil, err
	}
	response, err := h.scenarios().flash.GetReviewDays.Execute(ctx, v, req.From, req.To)
	if err != nil {
		return nil, newError(server.CodeInternal, err)
	}

	days := make([]*api.ReviewDay, 0, len(response.Days))
	for day, one := range response.Days {
		came := response.Retained[day]
		days = append(days, &api.ReviewDay{
			Day:      day,
			Reviewed: int32(one.Answered),
			Again:    int32(one.Again),
			Hard:     int32(one.Hard),
			Good:     int32(one.Good),
			Easy:     int32(one.Easy),
			Tested:   int32(came.Asked),
			Recalled: int32(came.Recalled),
		})
	}
	due := make([]*api.ReviewDay, 0, len(response.Due))
	for day, falls := range response.Due {
		due = append(due, &api.ReviewDay{
			Day: day, Reviewed: int32(falls), Again: 0, Hard: 0, Good: 0, Easy: 0, Tested: 0, Recalled: 0,
		})
	}

	return &api.ListReviewDaysResponse{
		Days:     sortReviewDays(days),
		Due:      sortReviewDays(due),
		Streak:   int32(response.Streak),
		Reviewed: int32(response.Answered),
	}, nil
}

// sortReviewDays puts the days oldest first.
func sortReviewDays(days []*api.ReviewDay) []*api.ReviewDay {
	slices.SortFunc(days, func(one, other *api.ReviewDay) int {
		return cmp.Compare(one.Day, other.Day)
	})
	return days
}

// GetDeckNeighbourhood returns what the selected deck is linked to.
//
// Nothing here writes and no run is named: a deck can be browsed before a
// session starts, and reading is not part of a session.
func (h *reviewHandler) GetDeckNeighbourhood(
	ctx context.Context, req *api.GetDeckNeighbourhoodRequest,
) (*api.GetDeckNeighbourhoodResponse, error) {
	v, err := h.getVault(ctx, req.Vault)
	if err != nil {
		return nil, err
	}
	joined, err := h.scenarios().neighbourhood.Execute(ctx, v, req.Deck)
	if err != nil {
		return nil, newError(server.CodeInternal, err)
	}

	out := &api.GetDeckNeighbourhoodResponse{
		Notes:  make([]*api.DeckNeighbour, 0, len(joined.Notes)),
		Unread: int32(joined.Unread),
	}
	for _, one := range joined.Notes {
		next := &api.DeckNeighbour{
			Address:     one.Written,
			Path:        one.Path,
			Title:       one.Title,
			Body:        one.Body,
			Label:       one.Label,
			IsPointedAt: !one.IsBacklink,
			IsAmbiguous: one.IsAmbiguous,
			Error:       api.ErrorCodeUnspecified,
		}
		if reason, ok := convertOutcomeToErrorCode(one.Outcome); ok {
			next.Error = reason
		}
		out.Notes = append(out.Notes, next)
	}
	return out, nil
}

// again is the interval at which a stream with no changes sends a message, so a page can tell an
// open stream from a closed one.
const again = time.Second

// WatchReloads signals that something changed, for as long as the caller is subscribed.
//
// The change itself is not carried: the window shows counts and the cards behind them, and they are
// requested again after any change.
func (h *reviewHandler) WatchReloads(
	ctx context.Context,
	_ *api.WatchReloadsRequest,
	send func(*api.WatchReloadsResponse) error,
) error {
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	moved, err := h.vault.WatchMoves(ctx)
	if err != nil {
		return newError(server.CodeUnavailable, err)
	}

	// The stream sends an initial message before any change. A stream silent until a file changes
	// is indistinguishable from one that never opened.
	if err := send(&api.WatchReloadsResponse{Reload: false}); err != nil {
		return err
	}

	repeat := clock.NewTicker(again)
	defer repeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-repeat.C:
			// No change, which the opening message also signals. A closed page fails the
			// write.
			if err := send(&api.WatchReloadsResponse{Reload: false}); err != nil {
				return err
			}
		case _, open := <-moved:
			if !open {
				return nil
			}
			h.scenarios().flash.ReviewCards.InvalidateAll()
			if err := send(&api.WatchReloadsResponse{Reload: true}); err != nil {
				return err
			}
		}
	}
}
