package hostendpoint

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/wire"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

const invalidConnection = v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_INVALID_CONNECTION

func (e *vaultEndpoint) GetConnection(
	ctx context.Context, req *connect.Request[v1.HostVaultServiceGetConnectionRequest],
) (*connect.Response[v1.HostVaultServiceGetConnectionResponse], error) {
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.HostVaultServiceGetConnectionResponse{Error: noVault}), nil
	}
	found, err := vault.GetConnection(ctx, req.Msg.GetFirst(), req.Msg.GetSecond(), req.Msg.GetType())
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.HostVaultServiceGetConnectionResponse{Error: code}), err
	}
	return connect.NewResponse(&v1.HostVaultServiceGetConnectionResponse{
		Connection: wire.ConvertConnectionToWire(found),
	}), nil
}

func (e *vaultEndpoint) SaveConnection(
	ctx context.Context, req *connect.Request[v1.HostVaultServiceSaveConnectionRequest],
) (*connect.Response[v1.HostVaultServiceSaveConnectionResponse], error) {
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.HostVaultServiceSaveConnectionResponse{Error: noVault}), nil
	}
	saving, convertErr := wire.ConvertConnectionFromWire(req.Msg.GetConnection())
	code := invalidConnection
	if convertErr == nil {
		code, err = answer(ctx, vault.SaveConnection(ctx, saving))
	}
	return connect.NewResponse(&v1.HostVaultServiceSaveConnectionResponse{Error: code}), err
}

func (e *vaultEndpoint) RemoveSide(
	ctx context.Context, req *connect.Request[v1.HostVaultServiceRemoveSideRequest],
) (*connect.Response[v1.HostVaultServiceRemoveSideResponse], error) {
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.HostVaultServiceRemoveSideResponse{Error: noVault}), nil
	}
	code, err := answer(ctx, vault.RemoveSide(ctx, req.Msg.GetPath(), req.Msg.GetTo(), req.Msg.GetType()))
	return connect.NewResponse(&v1.HostVaultServiceRemoveSideResponse{Error: code}), err
}

func (e *vaultEndpoint) RemoveConnection(
	ctx context.Context, req *connect.Request[v1.HostVaultServiceRemoveConnectionRequest],
) (*connect.Response[v1.HostVaultServiceRemoveConnectionResponse], error) {
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.HostVaultServiceRemoveConnectionResponse{Error: noVault}), nil
	}
	code, err := answer(ctx, vault.RemoveConnection(ctx, req.Msg.GetFirst(), req.Msg.GetSecond(), req.Msg.GetType()))
	return connect.NewResponse(&v1.HostVaultServiceRemoveConnectionResponse{Error: code}), err
}

func (e *vaultEndpoint) JoinNotes(
	ctx context.Context, req *connect.Request[v1.HostVaultServiceJoinNotesRequest],
) (*connect.Response[v1.HostVaultServiceJoinNotesResponse], error) {
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.HostVaultServiceJoinNotesResponse{Error: noVault}), nil
	}
	side, convertErr := wire.ConvertSideFromWire(req.Msg.GetSide())
	code := invalidConnection
	if convertErr == nil && side != nil {
		code, err = answer(ctx, vault.Join(ctx, req.Msg.GetFrom(), req.Msg.GetTo(), *side))
	}
	return connect.NewResponse(&v1.HostVaultServiceJoinNotesResponse{Error: code}), err
}

func (e *vaultEndpoint) CreateRelatedNote(
	ctx context.Context, req *connect.Request[v1.HostVaultServiceCreateRelatedNoteRequest],
) (*connect.Response[v1.HostVaultServiceCreateRelatedNoteResponse], error) {
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.HostVaultServiceCreateRelatedNoteResponse{Error: noVault}), nil
	}
	role, ok := wire.ConvertRoleFromWire(req.Msg.GetRole())
	if !ok {
		return connect.NewResponse(&v1.HostVaultServiceCreateRelatedNoteResponse{
			Error: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_INVALID_CONNECTION,
		}), nil
	}
	path, err := vault.CreateInRelation(ctx, domain.RelatedNote{
		From: req.Msg.GetFrom(), Role: role, Title: req.Msg.GetTitle(), Type: req.Msg.GetType(),
	})
	code, err := answer(ctx, err)
	return connect.NewResponse(&v1.HostVaultServiceCreateRelatedNoteResponse{Error: code, Path: path}), err
}
