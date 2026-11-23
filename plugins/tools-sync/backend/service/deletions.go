package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/deletions"
)

// WithDeletions gives the backend the answer to the pause before many removals.
func (b *Backend) WithDeletions(pending ports.PausedStore, resolve *deletions.ResolveDeletions) *Backend {
	b.pending, b.resolve = pending, resolve
	return b
}

func (b *Backend) getPausedRemovals(ctx context.Context, id domain.VaultID) (*api.PausedDeletions, error) {
	if b.pending == nil {
		return nil, nil
	}
	state, err := b.pending.ReadPending(ctx, id)
	if err != nil || state == nil {
		return nil, err
	}
	return &api.PausedDeletions{Count: len(state.Files), Tracked: state.Tracked, Direction: api.Direction(state.Direction)}, nil
}

// ResolveDeletions takes the answer of the person to the removals of a vault that wait for it.
func (b *Backend) ResolveDeletions(ctx context.Context, request api.ResolveDeletionsRequest) (api.ResolveDeletionsResponse, error) {
	if b.resolve == nil {
		return api.ResolveDeletionsResponse{}, deletions.ErrNothingPending
	}
	_, err := b.resolve.Execute(ctx, domain.VaultID(request.VaultID), transfer.Choice(request.Choice))
	return api.ResolveDeletionsResponse{}, err
}
