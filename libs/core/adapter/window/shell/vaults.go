package shell

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/notes/domain"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// errNoVaults is what a window built without the list of vaults answers.
var errNoVaults = errors.New("this window serves no list of vaults")

// vaultList returns the vaults the installation holds, for the user to choose
// from. The rest of VaultsService belongs to the editor.
type vaultList struct {
	numenv1connect.UnimplementedVaultsServiceHandler
	list func() ([]domain.Vault, error)
}

func (l vaultList) ListVaults(
	context.Context, *connect.Request[v1.ListVaultsRequest],
) (*connect.Response[v1.ListVaultsResponse], error) {
	if l.list == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoVaults)
	}
	target, err := l.list()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ListVaultsResponse{Vaults: make([]*v1.Vault, 0, len(target))}
	for _, v := range target {
		out.Vaults = append(out.Vaults, &v1.Vault{Id: string(v.ID), Name: v.Name, Path: v.Path})
	}
	return connect.NewResponse(out), nil
}
