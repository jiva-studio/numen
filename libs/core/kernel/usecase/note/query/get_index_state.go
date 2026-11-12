package query

import (
	"context"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// GetIndexState reports whether the index carries a vault yet: a vault the index holds is indexed, any
// other is still being read into it.
type GetIndexState struct {
	holds noteport.VaultHolds
}

func NewGetIndexState(holds noteport.VaultHolds) *GetIndexState {
	return &GetIndexState{holds: holds}
}

func (u *GetIndexState) Execute(ctx context.Context, v domain.Vault) (plugindomain.IndexState, error) {
	has, err := u.holds.Has(ctx, v.ID)
	if err != nil {
		return plugindomain.IndexState{}, err
	}
	if has {
		return plugindomain.IndexState{Status: plugindomain.Indexed}, nil
	}
	return plugindomain.IndexState{Status: plugindomain.Indexing}, nil
}
