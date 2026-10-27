package shell

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/notes/domain"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// errNoVault is what a request naming no vault is answered with: this window shows none.
var errNoVault = errors.New("this window shows no vault, so a request names the vault of its links")

// noteLinks answers where the links of a text land, in the vault the request names. The rest of
// NoteService belongs to the editor.
type noteLinks struct {
	numenv1connect.UnimplementedNoteServiceHandler
	resolve *note.ResolveLinks
	list    func() ([]domain.Vault, error)
}

func (l noteLinks) ResolveAddresses(
	ctx context.Context, r *connect.Request[v1.ResolveAddressesRequest],
) (*connect.Response[v1.ResolveAddressesResponse], error) {
	named := r.Msg.GetVault()
	if named == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNoVault)
	}
	if l.resolve == nil || l.list == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("this window resolves no links"))
	}
	vaults, err := l.list()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	for _, v := range vaults {
		if string(v.ID) != named {
			continue
		}
		out, err := wire.ResolveAddresses(ctx, l.resolve, v, r.Msg)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(out), nil
	}
	return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("no vault %q", named))
}
