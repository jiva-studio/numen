package service

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/apierror"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/flashcards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/adapter/clock"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/presetwire"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/reloads"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/reviewwire"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/vaultcount"
)

// again is the interval at which a stream with no changes sends a message, so a page can tell an
// open stream from a closed one.
const again = time.Second

// reviewHandler serves the review window.
//
// The runs it holds are the sessions open here, one per vault. A run is a file in a vault that only
// this backend appends to. It is released when another session opens on that vault: the written
// answers remain, and the next session opens a new file.
type reviewHandler struct {
	*Backend
}

func (h *reviewHandler) openRun(ctx context.Context, v domain.Vault) (*flashcards.LogWriter, error) {
	run, err := h.scenarios().Flash.Log.Open(ctx, v, h.now.Now())
	if err != nil {
		return nil, err
	}
	h.runs.Store(v, run)
	return run, nil
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
	createResponse := func() *api.WatchCardsDueResponse {
		return reviewwire.NewVaultsResponse(recognized, h.scenarios().Day.GetName(h.now.Now()))
	}
	if err := send(createResponse()); err != nil {
		return err
	}

	// A count left running is a vault read for a window that has gone, so the counters are ended
	// with this call whichever way it ends.
	ctx, stop := context.WithCancel(ctx)
	defer stop()

	// Nothing else signals this handler that its client has gone: a reloaded page does not cancel
	// the request context. A failed write is the only signal, so the list is sent again periodically
	// while nothing has been counted.
	repeat := clock.NewTicker(again)
	defer repeat.Stop()

	return vaultcount.Forward(vaultcount.Start(ctx, recognized, h.countVault), repeat.C,
		func(one *api.VaultCardsDue) error {
			if err := send(&api.WatchCardsDueResponse{Vaults: nil, Day: "", Counted: one}); err != nil {
				return err
			}
			repeat.Reset(again)
			return nil
		},
		func() error { return send(createResponse()) })
}

func (h *reviewHandler) countVault(ctx context.Context, v domain.Vault) *api.VaultCardsDue {
	counted := vaultcount.Count(ctx, h.vaults(v.ID).Index, h.scenarios().Flash.CardsDue, v)
	return reviewwire.NewVaultCardsDue(counted, h.newTitler(ctx, v))
}

// StartSession opens a session over one deck, over one preset, or over the whole vault, and
// returns the cards to ask in order.
func (h *reviewHandler) StartSession(ctx context.Context, req *api.StartSessionRequest) (*api.StartSessionResponse, error) {
	v, err := h.getVault(ctx, req.Vault)
	if err != nil {
		return nil, err
	}
	session, err := h.scenarios().Flash.Session.Execute(ctx, v, reviewwire.NewScope(req))
	if err != nil {
		return nil, reviewwire.NewSessionError(err)
	}
	run, err := h.openRun(ctx, v)
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
	run, err := h.runs.GetRun(v.ID, req.Run)
	if err != nil {
		return nil, reviewwire.NewRunError(err)
	}
	on := review.CardID{Card: req.CardID, Template: req.Template}
	answer, err := h.scenarios().Flash.Record.Answer(ctx, run, on, reviewwire.ParseRating(req.Rating),
		time.Duration(req.DurationMs)*time.Millisecond)
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
	run, err := h.runs.GetRun(v.ID, req.Run)
	if err != nil {
		return nil, reviewwire.NewRunError(err)
	}
	if _, err := h.scenarios().Flash.Record.TakeBack(ctx, run, req.Answer); err != nil {
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
	found, err := h.scenarios().Flash.Presets.GetForDeck(ctx, v, req.Deck)
	return presetwire.NewVaultDeckPresetResponse(found, err, h.newTitler(ctx, v))
}

func (h *reviewHandler) ListReviewDays(ctx context.Context, req *api.ListReviewDaysRequest) (*api.ListReviewDaysResponse, error) {
	v, err := h.getVault(ctx, req.Vault)
	if err != nil {
		return nil, err
	}
	counts, err := h.scenarios().Flash.GetReviewDays.Execute(ctx, v, req.From, req.To)
	return reviewwire.NewDaysResponse(counts, err)
}

// GetDeckNeighbourhood returns what the selected deck is linked to. Nothing here writes and no run
// is named: a deck can be browsed before a session starts.
func (h *reviewHandler) GetDeckNeighbourhood(
	ctx context.Context, req *api.GetDeckNeighbourhoodRequest,
) (*api.GetDeckNeighbourhoodResponse, error) {
	v, err := h.getVault(ctx, req.Vault)
	if err != nil {
		return nil, err
	}
	joined, err := h.scenarios().Neighborhood.Execute(ctx, v, req.Deck)
	return reviewwire.NewNeighbourhoodResponse(joined, err)
}

func (h *reviewHandler) WatchReloads(
	ctx context.Context, _ *api.WatchReloadsRequest, send func(*api.WatchReloadsResponse) error,
) error {
	repeat := clock.NewTicker(again)
	defer repeat.Stop()
	err := reloads.Watch(ctx, h.vaults("").Index, repeat.C,
		func() { h.scenarios().Flash.ReviewCards.InvalidateAll() },
		func(reload bool) error { return send(&api.WatchReloadsResponse{Reload: reload}) })
	return reviewwire.NewWatchError(err)
}
