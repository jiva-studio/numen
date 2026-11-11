package service

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/apierror"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/flashcards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/hostports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/neighborhood"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/presetwire"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/reloads"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/reviewwire"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/sessions"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/vaultcount"
)

// reviewHandler serves the review window.
type reviewHandler struct {
	vaults       func(domain.VaultID) hostports.Vault
	getVault     func(context.Context, string) (domain.Vault, error)
	newTitler    func(context.Context, domain.Vault) func(string) string
	now          ports.Clock
	runs         *sessions.Registry
	flash        func() *flashcards.Scenarios
	day          func() *review.Day
	neighborhood func() *neighborhood.Show
}

func (b *Backend) newReviewHandler() *reviewHandler {
	return &reviewHandler{
		vaults:       b.vaults,
		getVault:     b.getVault,
		newTitler:    b.newTitler,
		now:          b.now,
		runs:         b.runs,
		flash:        func() *flashcards.Scenarios { return &b.scenarios().Flash },
		day:          func() *review.Day { return &b.scenarios().Day },
		neighborhood: func() *neighborhood.Show { return &b.scenarios().Neighborhood },
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
		return send(reviewwire.NewVaultsResponse(recognized, h.day().GetName(h.now.Now())))
	}
	return vaultcount.Watch(ctx, recognized, h.countVault,
		func(one *api.VaultCardsDue) error {
			return send(&api.WatchCardsDueResponse{Vaults: nil, Day: "", Counted: one})
		},
		sendVaults)
}

func (h *reviewHandler) countVault(ctx context.Context, v domain.Vault) *api.VaultCardsDue {
	counted := vaultcount.Count(ctx, h.vaults(v.ID).Index, h.flash().CardsDue, v)
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
	session, err := flash.Session.Execute(ctx, v, reviewwire.NewScope(req))
	if err != nil {
		return nil, reviewwire.NewSessionError(err)
	}
	run, err := h.runs.Open(ctx, flash.Log, v, h.now.Now())
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
	answer, err := h.runs.RecordAnswer(ctx, h.flash().Record, sessions.AnswerRequest{
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
	if err := h.runs.UndoAnswer(ctx, h.flash().Record, v.ID, req.Run, req.Answer); err != nil {
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
	found, err := h.flash().Presets.GetForDeck(ctx, v, req.Deck)
	return presetwire.NewVaultDeckPresetResponse(found, err, h.newTitler(ctx, v))
}

func (h *reviewHandler) ListReviewDays(ctx context.Context, req *api.ListReviewDaysRequest) (*api.ListReviewDaysResponse, error) {
	v, err := h.getVault(ctx, req.Vault)
	if err != nil {
		return nil, err
	}
	counts, err := h.flash().GetReviewDays.Execute(ctx, v, req.From, req.To)
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
	joined, err := h.neighborhood().Execute(ctx, v, req.Deck)
	return reviewwire.NewNeighborhoodResponse(joined, err)
}

func (h *reviewHandler) WatchReloads(
	ctx context.Context, _ *api.WatchReloadsRequest, send func(*api.WatchReloadsResponse) error,
) error {
	err := reloads.WatchTicks(ctx, h.vaults("").Index,
		func() { h.flash().ReviewCards.InvalidateAll() },
		func(reload bool) error { return send(&api.WatchReloadsResponse{Reload: reload}) })
	return reviewwire.NewWatchError(err)
}
