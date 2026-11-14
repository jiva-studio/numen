package service

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/adapter/clock"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/flashcards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/neighborhood"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/note"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/reloads"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/vaultcount"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/wire/apierror"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/wire/presetwire"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/wire/reviewwire"
)

// reviewHandler serves the review window.
type reviewHandler struct {
	*vaultSession
	now          ports.Clock
	runs         *flashcards.RunRegistry
	flash        func() *flashReview
	neighborhood neighborhood.Show
}

func (b *Backend) newReviewHandler() *reviewHandler {
	return &reviewHandler{
		vaultSession: b.vaultSession,
		now:          b.now,
		runs:         b.runs,
		flash:        b.flash.Get,
		neighborhood: neighborhood.NewShow(note.NewShowLinks(b.ports.Links), b.ports.Notes, note.NewRead(b.ports.Readers)),
	}
}

// WatchCardsDue sends the vaults the installation contains, then the count of each as it is
// computed. A vault that cannot be counted is sent with no count and the error.
func (h *reviewHandler) WatchCardsDue(
	ctx context.Context, _ *api.WatchCardsDueRequest, send func(*api.WatchCardsDueResponse) error,
) error {
	recognized, err := h.vaults("").Selector.GetVaults(ctx)
	if err != nil {
		return apierror.NewInternal(err)
	}
	sendVaults := func() error {
		return send(reviewwire.NewVaultsResponse(recognized, h.flash().Day.GetName(h.now.Now())))
	}
	return vaultcount.Watch(ctx, recognized, clock.Tickers{}, h.countVault,
		func(one *api.VaultCardsDue) error {
			return send(&api.WatchCardsDueResponse{Vaults: nil, Day: "", Counted: one})
		},
		sendVaults)
}

func (h *reviewHandler) countVault(ctx context.Context, v domain.Vault) *api.VaultCardsDue {
	counted := vaultcount.Count(ctx, h.vaults(v.ID).Index, h.flash().Counts.CardsDue, v)
	return reviewwire.NewVaultCardsDue(counted, h.newTitler(ctx, v))
}

// StartSession opens a session over one deck, over one preset, or over the whole vault, and
// returns the cards to ask in order.
func (h *reviewHandler) StartSession(ctx context.Context, req *api.StartSessionRequest) (*api.StartSessionResponse, error) {
	v, err := h.getVault(ctx, req.Vault)
	if err != nil {
		return nil, err
	}
	flash := h.flash()
	session, err := flash.Reviews.Session.Execute(ctx, v, reviewwire.NewScope(req))
	if err != nil {
		return nil, reviewwire.NewSessionError(err)
	}
	run, err := h.runs.Open(ctx, flash.Reviews.Log, v, h.now.Now())
	if err != nil {
		return nil, apierror.NewInternal(err)
	}
	return reviewwire.NewSessionResponse(run.GetName(), session), nil
}

func (h *reviewHandler) RecordReview(ctx context.Context, req *api.RecordReviewRequest) (*api.RecordReviewResponse, error) {
	v, err := h.getVault(ctx, req.Vault)
	if err != nil {
		return nil, err
	}
	answer, err := h.runs.RecordAnswer(ctx, h.flash().Reviews.Record, flashcards.AnswerRequest{
		Vault:    v.ID,
		Run:      req.Run,
		Card:     review.CardID{Card: req.CardID, Template: req.Template},
		Rating:   reviewwire.ParseRating(req.Rating),
		Duration: time.Duration(req.DurationMs) * time.Millisecond,
	})
	if err != nil {
		return nil, reviewwire.NewRecordError(err)
	}
	return &api.RecordReviewResponse{Answer: answer.ID}, nil
}

func (h *reviewHandler) UndoReview(ctx context.Context, req *api.UndoReviewRequest) (*api.UndoReviewResponse, error) {
	v, err := h.getVault(ctx, req.Vault)
	if err != nil {
		return nil, err
	}
	if err := h.runs.UndoAnswer(ctx, h.flash().Reviews.Record, v.ID, req.Run, req.Answer); err != nil {
		return nil, reviewwire.NewUndoError(err)
	}
	return &api.UndoReviewResponse{}, nil
}

func (h *reviewHandler) GetVaultDeckPreset(
	ctx context.Context, req *api.GetVaultDeckPresetRequest,
) (*api.GetVaultDeckPresetResponse, error) {
	v, err := h.getVault(ctx, req.Vault)
	if err != nil {
		return nil, err
	}
	found, err := h.flash().Schedules.Presets.GetForDeck(ctx, v, req.Deck)
	return presetwire.NewVaultDeckPresetResponse(found, err, h.newTitler(ctx, v))
}

func (h *reviewHandler) ListReviewDays(ctx context.Context, req *api.ListReviewDaysRequest) (*api.ListReviewDaysResponse, error) {
	v, err := h.getVault(ctx, req.Vault)
	if err != nil {
		return nil, err
	}
	counts, err := h.flash().Counts.GetReviewDays.Execute(ctx, v, req.From, req.To)
	return reviewwire.NewDaysResponse(counts, err)
}

// GetDeckNeighborhood returns what the selected deck is linked to. Nothing here writes and no run
// is named: a deck can be browsed before a session starts.
func (h *reviewHandler) GetDeckNeighborhood(
	ctx context.Context, req *api.GetDeckNeighborhoodRequest,
) (*api.GetDeckNeighborhoodResponse, error) {
	v, err := h.getVault(ctx, req.Vault)
	if err != nil {
		return nil, err
	}
	joined, err := h.neighborhood.Execute(ctx, v, req.Deck)
	return reviewwire.NewNeighborhoodResponse(joined, err)
}

func (h *reviewHandler) WatchReloads(
	ctx context.Context, _ *api.WatchReloadsRequest, send func(*api.WatchReloadsResponse) error,
) error {
	err := reloads.WatchTicks(ctx, h.vaults("").Index, clock.Tickers{},
		func() { h.flash().Reviews.ReviewCards.InvalidateAll() },
		func(reload bool) error { return send(&api.WatchReloadsResponse{Reload: reload}) })
	return reviewwire.NewWatchError(err)
}
