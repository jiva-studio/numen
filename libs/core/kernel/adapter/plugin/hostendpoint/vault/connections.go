package vault

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/note/domain"
)

const invalidConnection = v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_INVALID_CONNECTION

func (e *connectionsCalls) GetConnection(
	ctx context.Context, req *connect.Request[v1.HostVaultServiceGetConnectionRequest],
) (*connect.Response[v1.HostVaultServiceGetConnectionResponse], error) {
	_, vault, err := e.begin(req.Header(), manifest.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.HostVaultServiceGetConnectionResponse{Error: noVault}), nil
	}
	found, err := vault.Connections.GetConnection(ctx, req.Msg.GetFirst(), req.Msg.GetSecond(), req.Msg.GetType())
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.HostVaultServiceGetConnectionResponse{Error: code}), err
	}
	return connect.NewResponse(&v1.HostVaultServiceGetConnectionResponse{
		Connection: wire.ConvertConnectionToWire(found),
	}), nil
}

func (e *connectionsCalls) SaveConnection(
	ctx context.Context, req *connect.Request[v1.HostVaultServiceSaveConnectionRequest],
) (*connect.Response[v1.HostVaultServiceSaveConnectionResponse], error) {
	_, vault, err := e.begin(req.Header(), manifest.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.HostVaultServiceSaveConnectionResponse{Error: noVault}), nil
	}
	saving, convertErr := wire.ConvertConnectionFromWire(req.Msg.GetConnection())
	code := invalidConnection
	if convertErr == nil {
		code, err = answer(ctx, vault.Connections.SaveConnection(ctx, saving))
	}
	return connect.NewResponse(&v1.HostVaultServiceSaveConnectionResponse{Error: code}), err
}

func (e *connectionsCalls) RemoveSide(
	ctx context.Context, req *connect.Request[v1.HostVaultServiceRemoveSideRequest],
) (*connect.Response[v1.HostVaultServiceRemoveSideResponse], error) {
	_, vault, err := e.begin(req.Header(), manifest.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.HostVaultServiceRemoveSideResponse{Error: noVault}), nil
	}
	code, err := answer(ctx, vault.Connections.RemoveSide(ctx, req.Msg.GetPath(), req.Msg.GetTo(), req.Msg.GetType()))
	return connect.NewResponse(&v1.HostVaultServiceRemoveSideResponse{Error: code}), err
}

func (e *connectionsCalls) RemoveConnection(
	ctx context.Context, req *connect.Request[v1.HostVaultServiceRemoveConnectionRequest],
) (*connect.Response[v1.HostVaultServiceRemoveConnectionResponse], error) {
	_, vault, err := e.begin(req.Header(), manifest.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.HostVaultServiceRemoveConnectionResponse{Error: noVault}), nil
	}
	code, err := answer(ctx, vault.Connections.RemoveConnection(ctx, req.Msg.GetFirst(), req.Msg.GetSecond(), req.Msg.GetType()))
	return connect.NewResponse(&v1.HostVaultServiceRemoveConnectionResponse{Error: code}), err
}

func (e *connectionsCalls) JoinNotes(
	ctx context.Context, req *connect.Request[v1.HostVaultServiceJoinNotesRequest],
) (*connect.Response[v1.HostVaultServiceJoinNotesResponse], error) {
	_, vault, err := e.begin(req.Header(), manifest.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.HostVaultServiceJoinNotesResponse{Error: noVault}), nil
	}
	side, convertErr := wire.ConvertSideFromWire(req.Msg.GetSide())
	code := invalidConnection
	if convertErr == nil && side != nil {
		code, err = answer(ctx, vault.Connections.Join(ctx, req.Msg.GetFrom(), req.Msg.GetTo(), *side))
	}
	return connect.NewResponse(&v1.HostVaultServiceJoinNotesResponse{Error: code}), err
}

func (e *connectionsCalls) CreateRelatedNote(
	ctx context.Context, req *connect.Request[v1.HostVaultServiceCreateRelatedNoteRequest],
) (*connect.Response[v1.HostVaultServiceCreateRelatedNoteResponse], error) {
	_, vault, err := e.begin(req.Header(), manifest.PermissionVaultWrite)
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
	marker, ok := wire.ConvertMarkerFromWire(req.Msg.GetMarker())
	if !ok {
		return connect.NewResponse(&v1.HostVaultServiceCreateRelatedNoteResponse{
			Error: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_INVALID_CONNECTION,
		}), nil
	}
	path, err := vault.Related.CreateInRelation(ctx, domain.RelatedNote{
		From: req.Msg.GetFrom(), Role: role, Title: req.Msg.GetTitle(), Type: req.Msg.GetType(), Marker: marker,
	})
	code, err := answer(ctx, err)
	return connect.NewResponse(&v1.HostVaultServiceCreateRelatedNoteResponse{Error: code, Path: path}), err
}
