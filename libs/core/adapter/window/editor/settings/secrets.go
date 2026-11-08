package settings

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/usecase/secret"
)

func newSecretError(err error) error {
	switch {
	case errors.Is(err, secret.ErrInvalid):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, secret.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, secret.ErrKeyStore):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}

func (h *Handler) checkSecrets() error {
	if h.Secrets == nil {
		return connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	return nil
}

// CreateSecret adds a secret at the end of its group and keeps its value in the key store.
func (h *Handler) CreateSecret(
	ctx context.Context, r *connect.Request[v1.CreateSecretRequest],
) (*connect.Response[v1.CreateSecretResponse], error) {
	if err := h.checkSecrets(); err != nil {
		return nil, err
	}
	id, err := h.Secrets.Create.Execute(ctx, r.Msg.GetGroup(), r.Msg.GetName(), r.Msg.GetValue())
	if err != nil {
		return nil, newSecretError(err)
	}
	return connect.NewResponse(&v1.CreateSecretResponse{Id: id}), nil
}

// RenameSecret changes the name of a secret.
func (h *Handler) RenameSecret(
	ctx context.Context, r *connect.Request[v1.RenameSecretRequest],
) (*connect.Response[v1.RenameSecretResponse], error) {
	if err := h.checkSecrets(); err != nil {
		return nil, err
	}
	if err := h.Secrets.Rename.Execute(ctx, r.Msg.GetId(), r.Msg.GetName()); err != nil {
		return nil, newSecretError(err)
	}
	return connect.NewResponse(&v1.RenameSecretResponse{}), nil
}

// WriteSecret replaces the value of a secret.
func (h *Handler) WriteSecret(
	ctx context.Context, r *connect.Request[v1.WriteSecretRequest],
) (*connect.Response[v1.WriteSecretResponse], error) {
	if err := h.checkSecrets(); err != nil {
		return nil, err
	}
	if err := h.Secrets.Write.Execute(ctx, r.Msg.GetId(), r.Msg.GetValue()); err != nil {
		return nil, newSecretError(err)
	}
	return connect.NewResponse(&v1.WriteSecretResponse{}), nil
}

// DeleteSecret removes a secret with its value and its permissions.
func (h *Handler) DeleteSecret(
	ctx context.Context, r *connect.Request[v1.DeleteSecretRequest],
) (*connect.Response[v1.DeleteSecretResponse], error) {
	if err := h.checkSecrets(); err != nil {
		return nil, err
	}
	if err := h.Secrets.Delete.Execute(ctx, r.Msg.GetId()); err != nil {
		return nil, newSecretError(err)
	}
	return connect.NewResponse(&v1.DeleteSecretResponse{}), nil
}

// MoveSecret changes the priority of a secret inside its group.
func (h *Handler) MoveSecret(
	ctx context.Context, r *connect.Request[v1.MoveSecretRequest],
) (*connect.Response[v1.MoveSecretResponse], error) {
	if err := h.checkSecrets(); err != nil {
		return nil, err
	}
	if err := h.Secrets.Move.Execute(ctx, r.Msg.GetId(), r.Msg.GetBeforeId()); err != nil {
		return nil, newSecretError(err)
	}
	return connect.NewResponse(&v1.MoveSecretResponse{}), nil
}

// RenameSecretGroup changes the group of all the secrets of a group the person made.
func (h *Handler) RenameSecretGroup(
	ctx context.Context, r *connect.Request[v1.RenameSecretGroupRequest],
) (*connect.Response[v1.RenameSecretGroupResponse], error) {
	if err := h.checkSecrets(); err != nil {
		return nil, err
	}
	if err := h.Secrets.RenameGroup.Execute(ctx, r.Msg.GetGroup(), r.Msg.GetName()); err != nil {
		return nil, newSecretError(err)
	}
	return connect.NewResponse(&v1.RenameSecretGroupResponse{}), nil
}

// AllowPluginSecret lets one declaration of a plugin read a secret.
func (h *Handler) AllowPluginSecret(
	ctx context.Context, r *connect.Request[v1.AllowPluginSecretRequest],
) (*connect.Response[v1.AllowPluginSecretResponse], error) {
	if err := h.checkSecrets(); err != nil {
		return nil, err
	}
	permission := secret.Permission{Plugin: r.Msg.GetPluginId(), Name: r.Msg.GetName(), Group: r.Msg.GetGroup()}
	if err := h.Secrets.Allow.Execute(ctx, r.Msg.GetSecretId(), permission); err != nil {
		return nil, newSecretError(err)
	}
	return connect.NewResponse(&v1.AllowPluginSecretResponse{}), nil
}

// DenyPluginSecret stops one declaration of a plugin from reading a secret.
func (h *Handler) DenyPluginSecret(
	ctx context.Context, r *connect.Request[v1.DenyPluginSecretRequest],
) (*connect.Response[v1.DenyPluginSecretResponse], error) {
	if err := h.checkSecrets(); err != nil {
		return nil, err
	}
	permission := secret.Permission{Plugin: r.Msg.GetPluginId(), Name: r.Msg.GetName(), Group: r.Msg.GetGroup()}
	if err := h.Secrets.Deny.Execute(ctx, r.Msg.GetSecretId(), permission); err != nil {
		return nil, newSecretError(err)
	}
	return connect.NewResponse(&v1.DenyPluginSecretResponse{}), nil
}

// DeleteSecretGroup removes every secret of a group with its values and its permissions.
func (h *Handler) DeleteSecretGroup(
	ctx context.Context, r *connect.Request[v1.DeleteSecretGroupRequest],
) (*connect.Response[v1.DeleteSecretGroupResponse], error) {
	if err := h.checkSecrets(); err != nil {
		return nil, err
	}
	if err := h.Secrets.DeleteGroup.Execute(ctx, r.Msg.GetGroup()); err != nil {
		return nil, newSecretError(err)
	}
	return connect.NewResponse(&v1.DeleteSecretGroupResponse{}), nil
}

// ListSecretStates tells for each secret whether its value is on this machine.
func (h *Handler) ListSecretStates(
	ctx context.Context, _ *connect.Request[v1.ListSecretStatesRequest],
) (*connect.Response[v1.ListSecretStatesResponse], error) {
	if err := h.checkSecrets(); err != nil {
		return nil, err
	}
	states, err := h.Secrets.ListStates.Execute(ctx)
	if err != nil {
		return nil, newSecretError(err)
	}
	wire := make([]*v1.SecretState, 0, len(states))
	for _, one := range states {
		wire = append(wire, &v1.SecretState{Id: one.ID, HasValue: one.HasValue})
	}
	return connect.NewResponse(&v1.ListSecretStatesResponse{States: wire}), nil
}

func newWireSecretSections(sections []secret.Section) []*v1.SecretSection {
	wire := make([]*v1.SecretSection, 0, len(sections))
	for _, one := range sections {
		wire = append(wire, &v1.SecretSection{Group: one.Group, IsAsked: one.IsAsked, Secrets: newWireStoredSecrets(one.Secrets)})
	}
	return wire
}

func newWireSecretMenus(menus []secret.Menu) []*v1.SecretMenu {
	wire := make([]*v1.SecretMenu, 0, len(menus))
	for _, one := range menus {
		ids := make([]string, 0, len(one.Secrets))
		for _, entry := range one.Secrets {
			ids = append(ids, entry.ID)
		}
		wire = append(wire, &v1.SecretMenu{
			PluginId: one.Declaration.Plugin, Name: one.Declaration.Name, Group: one.Declaration.Group, SecretIds: ids,
		})
	}
	return wire
}

func newWireStoredSecrets(list []secret.Secret) []*v1.Secret {
	secrets := make([]*v1.Secret, 0, len(list))
	for _, one := range list {
		allowed := make([]*v1.SecretPermission, 0, len(one.Allowed))
		for _, permission := range one.Allowed {
			allowed = append(allowed, &v1.SecretPermission{
				PluginId: permission.Plugin, Name: permission.Name, Group: permission.Group,
			})
		}
		secrets = append(secrets, &v1.Secret{Id: one.ID, Name: one.Name, Group: one.Group, Allowed: allowed})
	}
	return secrets
}
