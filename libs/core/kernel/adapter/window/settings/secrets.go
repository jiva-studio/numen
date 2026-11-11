package settings

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/secret"
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

// SecretItems answers the calls that change one secret. A value of it with no use case set answers
// that the build serves no secrets.
type SecretItems struct {
	create *secret.Create
	rename *secret.Rename
	write  *secret.Write
	delete *secret.Delete
	move   *secret.Move
}

// NewSecretItems is the part of SettingsService that changes one secret.
func NewSecretItems(
	create *secret.Create, rename *secret.Rename, write *secret.Write, remove *secret.Delete, move *secret.Move,
) SecretItems {
	return SecretItems{create: create, rename: rename, write: write, delete: remove, move: move}
}

func (h *SecretItems) check() error {
	if h.create == nil {
		return connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	return nil
}

// SecretGroups answers the calls that change the groups of secrets and who may read them. A value
// of it with no use case set answers that the build serves no secrets.
type SecretGroups struct {
	renameGroup *secret.RenameGroup
	deleteGroup *secret.DeleteGroup
	allow       *secret.Allow
	deny        *secret.Deny
	listStates  *secret.ListStates
}

// NewSecretGroups is the part of SettingsService that changes the groups of secrets and who may read them.
func NewSecretGroups(
	rename *secret.RenameGroup, remove *secret.DeleteGroup, allow *secret.Allow, deny *secret.Deny, listStates *secret.ListStates,
) SecretGroups {
	return SecretGroups{renameGroup: rename, deleteGroup: remove, allow: allow, deny: deny, listStates: listStates}
}

func (h *SecretGroups) check() error {
	if h.listStates == nil {
		return connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	return nil
}

// CreateSecret adds a secret at the end of its group and keeps its value in the key store.
func (h *SecretItems) CreateSecret(
	ctx context.Context, r *connect.Request[v1.CreateSecretRequest],
) (*connect.Response[v1.CreateSecretResponse], error) {
	if err := h.check(); err != nil {
		return nil, err
	}
	id, err := h.create.Execute(ctx, r.Msg.GetGroup(), r.Msg.GetName(), r.Msg.GetValue())
	if err != nil {
		return nil, newSecretError(err)
	}
	return connect.NewResponse(&v1.CreateSecretResponse{Id: id}), nil
}

// RenameSecret changes the name of a secret.
func (h *SecretItems) RenameSecret(
	ctx context.Context, r *connect.Request[v1.RenameSecretRequest],
) (*connect.Response[v1.RenameSecretResponse], error) {
	if err := h.check(); err != nil {
		return nil, err
	}
	if err := h.rename.Execute(ctx, r.Msg.GetId(), r.Msg.GetName()); err != nil {
		return nil, newSecretError(err)
	}
	return connect.NewResponse(&v1.RenameSecretResponse{}), nil
}

// WriteSecret replaces the value of a secret.
func (h *SecretItems) WriteSecret(
	ctx context.Context, r *connect.Request[v1.WriteSecretRequest],
) (*connect.Response[v1.WriteSecretResponse], error) {
	if err := h.check(); err != nil {
		return nil, err
	}
	if err := h.write.Execute(ctx, r.Msg.GetId(), r.Msg.GetValue()); err != nil {
		return nil, newSecretError(err)
	}
	return connect.NewResponse(&v1.WriteSecretResponse{}), nil
}

// DeleteSecret removes a secret with its value and its permissions.
func (h *SecretItems) DeleteSecret(
	ctx context.Context, r *connect.Request[v1.DeleteSecretRequest],
) (*connect.Response[v1.DeleteSecretResponse], error) {
	if err := h.check(); err != nil {
		return nil, err
	}
	if err := h.delete.Execute(ctx, r.Msg.GetId()); err != nil {
		return nil, newSecretError(err)
	}
	return connect.NewResponse(&v1.DeleteSecretResponse{}), nil
}

// MoveSecret changes the priority of a secret inside its group.
func (h *SecretItems) MoveSecret(
	ctx context.Context, r *connect.Request[v1.MoveSecretRequest],
) (*connect.Response[v1.MoveSecretResponse], error) {
	if err := h.check(); err != nil {
		return nil, err
	}
	if err := h.move.Execute(ctx, r.Msg.GetId(), r.Msg.GetBeforeId()); err != nil {
		return nil, newSecretError(err)
	}
	return connect.NewResponse(&v1.MoveSecretResponse{}), nil
}

// RenameSecretGroup changes the group name for all secrets in a custom group.
func (h *SecretGroups) RenameSecretGroup(
	ctx context.Context, r *connect.Request[v1.RenameSecretGroupRequest],
) (*connect.Response[v1.RenameSecretGroupResponse], error) {
	if err := h.check(); err != nil {
		return nil, err
	}
	if err := h.renameGroup.Execute(ctx, r.Msg.GetGroup(), r.Msg.GetName()); err != nil {
		return nil, newSecretError(err)
	}
	return connect.NewResponse(&v1.RenameSecretGroupResponse{}), nil
}

// AllowPluginSecret lets one declaration of a plugin read a secret.
func (h *SecretGroups) AllowPluginSecret(
	ctx context.Context, r *connect.Request[v1.AllowPluginSecretRequest],
) (*connect.Response[v1.AllowPluginSecretResponse], error) {
	if err := h.check(); err != nil {
		return nil, err
	}
	permission := plugin.DeclarationKey{Plugin: r.Msg.GetPluginId(), Name: r.Msg.GetName(), Group: r.Msg.GetGroup()}
	if err := h.allow.Execute(ctx, r.Msg.GetSecretId(), permission); err != nil {
		return nil, newSecretError(err)
	}
	return connect.NewResponse(&v1.AllowPluginSecretResponse{}), nil
}

// DenyPluginSecret stops one declaration of a plugin from reading a secret.
func (h *SecretGroups) DenyPluginSecret(
	ctx context.Context, r *connect.Request[v1.DenyPluginSecretRequest],
) (*connect.Response[v1.DenyPluginSecretResponse], error) {
	if err := h.check(); err != nil {
		return nil, err
	}
	permission := plugin.DeclarationKey{Plugin: r.Msg.GetPluginId(), Name: r.Msg.GetName(), Group: r.Msg.GetGroup()}
	if err := h.deny.Execute(ctx, r.Msg.GetSecretId(), permission); err != nil {
		return nil, newSecretError(err)
	}
	return connect.NewResponse(&v1.DenyPluginSecretResponse{}), nil
}

// DeleteSecretGroup removes every secret of a group with its values and its permissions.
func (h *SecretGroups) DeleteSecretGroup(
	ctx context.Context, r *connect.Request[v1.DeleteSecretGroupRequest],
) (*connect.Response[v1.DeleteSecretGroupResponse], error) {
	if err := h.check(); err != nil {
		return nil, err
	}
	if err := h.deleteGroup.Execute(ctx, r.Msg.GetGroup()); err != nil {
		return nil, newSecretError(err)
	}
	return connect.NewResponse(&v1.DeleteSecretGroupResponse{}), nil
}

// ListSecretStates reports for each secret whether its value is stored on this machine.
func (h *SecretGroups) ListSecretStates(
	ctx context.Context, _ *connect.Request[v1.ListSecretStatesRequest],
) (*connect.Response[v1.ListSecretStatesResponse], error) {
	if err := h.check(); err != nil {
		return nil, err
	}
	states, err := h.listStates.Execute(ctx)
	if err != nil {
		return nil, newSecretError(err)
	}
	wire := make([]*v1.SecretState, 0, len(states))
	for _, one := range states {
		wire = append(wire, &v1.SecretState{Id: one.ID, HasValue: one.HasValue})
	}
	return connect.NewResponse(&v1.ListSecretStatesResponse{States: wire}), nil
}

func newWireSecretSections(sections []plugin.SecretSection) []*v1.SecretSection {
	wire := make([]*v1.SecretSection, 0, len(sections))
	for _, one := range sections {
		wire = append(wire, &v1.SecretSection{Group: one.Group, IsAsked: one.IsAsked, Secrets: newWireStoredSecrets(one.Secrets)})
	}
	return wire
}

func newWireSecretMenus(menus []plugin.SecretMenu) []*v1.SecretMenu {
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

func newWireStoredSecrets(list []plugin.Secret) []*v1.Secret {
	secrets := make([]*v1.Secret, 0, len(list))
	for _, one := range list {
		allowed := make([]*v1.DeclarationKey, 0, len(one.Allowed))
		for _, permission := range one.Allowed {
			allowed = append(allowed, &v1.DeclarationKey{
				PluginId: permission.Plugin, Name: permission.Name, Group: permission.Group,
			})
		}
		secrets = append(secrets, &v1.Secret{Id: one.ID, Name: one.Name, Group: one.Group, Allowed: allowed})
	}
	return secrets
}
