package shell

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// errNoVault is what a request naming no vault is answered with: this window shows none.
var errNoVault = errors.New("this window shows no vault, so a request names the vault of its links")

// noteLinks answers where the links of a text land, in the vault the request names. The rest of
// NoteService belongs to the editor.
type noteLinks struct {
	numenv1connect.UnimplementedNoteServiceHandler
	resolve *query.ResolveVaultLinks
}

func (l noteLinks) ResolveAddresses(
	ctx context.Context, r *connect.Request[v1.ResolveAddressesRequest],
) (*connect.Response[v1.ResolveAddressesResponse], error) {
	named := r.Msg.GetVault()
	if named == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNoVault)
	}
	if l.resolve == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("this window resolves no links"))
	}
	out, err := wire.ResolveAddresses(ctx, l.resolve, domain.VaultID(named), r.Msg)
	if errors.Is(err, query.ErrUnknownVault) {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}
