package note

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// GetResolvedAddresses finds where addresses written in one note land.
type GetResolvedAddresses struct {
	links port.LinkQueries
}

func NewGetResolvedAddresses(links port.LinkQueries) GetResolvedAddresses {
	return GetResolvedAddresses{links: links}
}

// Execute answers by what was written. An address that reaches nothing is absent.
func (u GetResolvedAddresses) Execute(
	ctx context.Context, v domain.Vault, from string, written []string,
) (map[string]domain.ResolvedLink, error) {
	return u.links.ResolveLinks(ctx, v.ID, from, written)
}
