package hostendpoint

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
)

// errNotServed is what the calls that rename, move and remove notes, change the list of vaults and
// run the checks of a vault answer until they are served.
var errNotServed = errors.New("hostendpoint: not implemented")

func (e *vaultEndpoint) GetVaultProblems(
	context.Context, *connect.Request[v1.GetVaultProblemsRequest],
) (*connect.Response[v1.GetVaultProblemsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errNotServed)
}

func (e *vaultEndpoint) RenameNote(
	context.Context, *connect.Request[v1.HostVaultServiceRenameNoteRequest],
) (*connect.Response[v1.HostVaultServiceRenameNoteResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errNotServed)
}

func (e *vaultEndpoint) MoveNotes(
	context.Context, *connect.Request[v1.HostVaultServiceMoveNotesRequest],
) (*connect.Response[v1.HostVaultServiceMoveNotesResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errNotServed)
}

func (e *vaultEndpoint) RemoveNotes(
	context.Context, *connect.Request[v1.HostVaultServiceRemoveNotesRequest],
) (*connect.Response[v1.HostVaultServiceRemoveNotesResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errNotServed)
}

func (e *vaultEndpoint) AddVault(
	context.Context, *connect.Request[v1.HostVaultServiceAddVaultRequest],
) (*connect.Response[v1.HostVaultServiceAddVaultResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errNotServed)
}

func (e *vaultEndpoint) RenameVault(
	context.Context, *connect.Request[v1.HostVaultServiceRenameVaultRequest],
) (*connect.Response[v1.HostVaultServiceRenameVaultResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errNotServed)
}

func (e *vaultEndpoint) RemoveVault(
	context.Context, *connect.Request[v1.HostVaultServiceRemoveVaultRequest],
) (*connect.Response[v1.HostVaultServiceRemoveVaultResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errNotServed)
}

func (e *vaultEndpoint) OpenVault(
	context.Context, *connect.Request[v1.HostVaultServiceOpenVaultRequest],
) (*connect.Response[v1.HostVaultServiceOpenVaultResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errNotServed)
}
