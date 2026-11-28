package share

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// CloseShares closes every open share of a vault.
type CloseShares struct{ deps Deps }

// NewCloseShares returns the use case over the dependencies.
func NewCloseShares(deps Deps) *CloseShares { return &CloseShares{deps: deps} }

// Execute appends the entry that closes each share of the vault that is open. A vault with no share, or one that is not
// connected, has nothing to close.
func (u *CloseShares) Execute(ctx context.Context, id domain.VaultID) error {
	roots, err := NewReadShares(u.deps).Execute(ctx, id)
	if err != nil {
		return err
	}
	stop := NewStopShare(u.deps)
	for _, root := range roots {
		if root.IsClosed {
			continue
		}
		if err := stop.Execute(ctx, id, root.Folder); err != nil {
			return err
		}
	}
	return nil
}
