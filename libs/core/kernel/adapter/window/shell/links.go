package shell

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// errNoVaultNamed is what a request naming no vault is answered with.
var errNoVaultNamed = errors.New("the request names no vault")

// noteLinks answers where the links of a text land, in the vault the request names. The rest of
// NoteService belongs to the editor.
type noteLinks struct {
	numenv1connect.UnimplementedNoteServiceHandler
	resolve *query.ResolveVaultLinks
}

func (l noteLinks) ResolveAddresses(
	ctx context.Context, r *connect.Request[v1.ResolveAddressesRequest],
) (*connect.Response[v1.ResolveAddressesResponse], error) {
	named := r.Msg.GetVaultId()
	if named == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNoVaultNamed)
	}
	out, err := wire.ResolveAddresses(ctx, l.resolve, domain.VaultID(named), r.Msg)
	if errors.Is(err, query.ErrUnknownVault) {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err != nil {
		return nil, wire.NewInternalError(err)
	}
	return connect.NewResponse(out), nil
}
