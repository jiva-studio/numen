package note

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// GetLinkAddress finds how a link names the note at a path.
type GetLinkAddress struct {
	names NameQueries
}

func NewGetLinkAddress(names NameQueries) *GetLinkAddress {
	return &GetLinkAddress{names: names}
}

// Execute writes the name of the note where no index can say what else is filed
// under it.
func (u *GetLinkAddress) Execute(ctx context.Context, v domain.Vault, path string) (domain.Address, error) {
	if u == nil || u.names == nil {
		return domain.Address{Scheme: domain.SchemeName, Value: domain.Basename(path)}, nil
	}
	return GetAddress(ctx, u.names, v.ID, path)
}
