// Package reviewwire converts what the review window asks and is answered with, between the
// schema and the types of the use cases.
package reviewwire

import (
	"cmp"
	"errors"
	"slices"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/apierror"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/flashcards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/presetwire"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/reloads"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/sessions"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/vaultcount"
)

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(review.TimestampLayout)
}

// NewVaultHeader is a vault named with no count.
func NewVaultHeader(v domain.Vault) *api.VaultCardsDue {
	return &api.VaultCardsDue{
		ID: string(v.ID), Name: v.Name, Path: v.Path,
		ReviewCards: 0, Due: 0, New: 0, Decks: nil, Presets: nil, Unread: "", IsIndexing: false,
	}
}

// NewVaultCardsDue is the count of a vault. The title gives the name of a preset by its path.
func NewVaultCardsDue(counted vaultcount.Result, title func(path string) string) *api.VaultCardsDue {
	one := NewVaultHeader(counted.Vault)
	one.IsIndexing = counted.IsIndexing
	one.Unread = counted.Unread
	if counted.IsIndexing || counted.Unread != "" {
		return one
	}
	counts := counted.Counts
	one.ReviewCards = int32(counts.ReviewCards)
	one.Due = int32(counts.Due)
	one.New = int32(counts.New)
	for _, deck := range counts.Decks {
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
	for _, preset := range counts.Presets {
		one.Presets = append(one.Presets, &api.PresetCardsDue{
			Preset:          preset.Preset,
			Title:           title(preset.Preset),
			Decks:           int32(preset.Decks),
			Cards:           int32(preset.Cards),
			PendingReviews:  int32(preset.Due),
			PendingNew:      int32(preset.New),
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

// NewScope is the cards a session is asked over.
func NewScope(req *api.StartSessionRequest) flashcards.Scope {
	scope := flashcards.Scope{Deck: req.Deck, Preset: "", IsNamed: req.Preset != nil}
	if req.Preset != nil {
		scope.Preset = *req.Preset
	}
	return scope
}

// NewSessionError is why a session could not be opened, as an API error.
func NewSessionError(err error) error {
	switch {
	case errors.Is(err, flashcards.ErrBothNamed):
		return apierror.NewError(server.CodeInvalidArgument, err)
	case errors.Is(err, flashcards.ErrNotCarried),
		errors.Is(err, flashcards.ErrSchedulesNothing):
		return apierror.NewError(server.CodeFailedPrecondition, err)
	}
	return apierror.NewError(server.CodeInternal, err)
}

// NewSessionResponse is the cards of a session, in the order they are asked, and the run their answers go to.
func NewSessionResponse(run string, session flashcards.SessionResult) *api.StartSessionResponse {
	out := &api.StartSessionResponse{
		Run:       run,
		Cards:     make([]*api.SessionCard, 0, len(session.Queue)),
		Unwritten: session.Unwritten,
		Skipped:   int32(session.Skipped),
	}
	for _, one := range session.Queue {
		out.Cards = append(out.Cards, newSessionCard(one))
	}
	return out
}

func newSessionCard(one flashcards.QueuedReviewCard) *api.SessionCard {
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
		Intervals: newIntervals(one.Ahead),
	}
}

func newIntervals(ahead map[review.Rating]time.Duration) *api.Intervals {
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

// ParseRating is the rating an answer names, and zero for one it does not.
func ParseRating(r api.Rating) review.Rating {
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
		// gomutants:disable-next-line BRANCH_CASE reason="Unspecified rating shares the default zero return."
		return 0
	default:
		return 0
	}
}

// NewRecordError is why an answer could not be written down, as an API error.
func NewRecordError(err error) error {
	if errors.Is(err, sessions.ErrNoRun) {
		return NewRunError(err)
	}
	if errors.Is(err, flashcards.ErrNoRating) {
		return apierror.NewError(server.CodeInvalidArgument, err)
	}
	return apierror.NewError(server.CodeInternal, err)
}

// NewUndoError is why an answer could not be taken back, as an API error.
func NewUndoError(err error) error {
	if errors.Is(err, sessions.ErrNoRun) {
		return NewRunError(err)
	}
	if errors.Is(err, flashcards.ErrNoAnswer) {
		return apierror.NewError(server.CodeInvalidArgument, err)
	}
	return apierror.NewError(server.CodeInternal, err)
}

// NewDaysResponse is how much of a vault was answered on each day it was reviewed, oldest first.
func NewDaysResponse(response flashcards.ReviewCounts, err error) (*api.ListReviewDaysResponse, error) {
	if err != nil {
		return nil, apierror.NewInternal(err)
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
			Tested:   int32(came.Tested),
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

func sortReviewDays(days []*api.ReviewDay) []*api.ReviewDay {
	slices.SortFunc(days, func(one, other *api.ReviewDay) int {
		return cmp.Compare(one.Day, other.Day)
	})
	return days
}

// NewNeighborhoodResponse is what a deck is linked to.
func NewNeighborhoodResponse(joined flashcards.Neighborhood, err error) (*api.GetDeckNeighborhoodResponse, error) {
	if err != nil {
		return nil, apierror.NewInternal(err)
	}
	out := &api.GetDeckNeighborhoodResponse{
		Notes:  make([]*api.DeckNeighbor, 0, len(joined.Notes)),
		Unread: int32(joined.Unread),
	}
	for _, one := range joined.Notes {
		next := &api.DeckNeighbor{
			Address:     one.Written,
			Path:        one.Path,
			Title:       one.Title,
			Body:        one.Body,
			Label:       one.Label,
			IsPointedAt: !one.IsBacklink,
			IsAmbiguous: one.IsAmbiguous,
			Error:       api.ErrorCodeUnspecified,
		}
		if reason, ok := apierror.ConvertOutcome(one.Outcome); ok {
			next.Error = reason
		}
		out.Notes = append(out.Notes, next)
	}
	return out, nil
}

// NewWatchError is the end of a watch as an API error: a failure to follow the changes is
// unavailable, and a listener that has gone is returned as it came.
func NewWatchError(err error) error {
	if down, ok := errors.AsType[*reloads.UnavailableError](err); ok {
		return apierror.NewError(server.CodeUnavailable, down.Err)
	}
	return err
}

// NewRunError is a run that is not the one open on the vault, as an API error.
func NewRunError(err error) error { return apierror.NewError(server.CodeNotFound, err) }

// NewVaultsResponse is the vaults of the installation, named and not yet counted, and the day the
// counts to come stand in, which is the day a goal is weighed against.
func NewVaultsResponse(vaults []domain.Vault, day string) *api.WatchCardsDueResponse {
	items := make([]*api.VaultCardsDue, 0, len(vaults))
	for _, one := range vaults {
		items = append(items, NewVaultHeader(one))
	}
	return &api.WatchCardsDueResponse{Vaults: items, Day: day, Counted: nil}
}
