package host

import (
	"context"
	"fmt"

	notesdomain "github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
)

var _ VaultList = (*Client)(nil)

// failVaults converts the error code of a change of the list of vaults into its error, nil when
// there is none. A refusal carries the core's text in detail.
func failVaults(call string, code v1.VaultsErrorCode, detail string) error {
	err := readVaultsError(code)
	if err == nil {
		return nil
	}
	if detail != "" {
		return fmt.Errorf("%s: %w: %s", call, err, detail)
	}
	return fmt.Errorf("%s: %w", call, err)
}

func readVaultsError(code v1.VaultsErrorCode) error {
	switch code {
	case v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNSPECIFIED:
		return nil
	case v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNREADABLE:
		return ErrVaultUnreadable
	case v1.VaultsErrorCode_VAULTS_ERROR_CODE_COPY:
		return ErrVaultCopy
	case v1.VaultsErrorCode_VAULTS_ERROR_CODE_OVERLAPS:
		return ErrVaultOverlaps
	case v1.VaultsErrorCode_VAULTS_ERROR_CODE_NAME_TAKEN:
		return ErrVaultNameTaken
	case v1.VaultsErrorCode_VAULTS_ERROR_CODE_LAST_VAULT:
		return ErrLastVault
	case v1.VaultsErrorCode_VAULTS_ERROR_CODE_CURRENT:
		return ErrVaultShowing
	case v1.VaultsErrorCode_VAULTS_ERROR_CODE_NOT_FOUND:
		return ErrUnknownVault
	case v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNSAVED:
		return ErrUnsavedChanges
	default:
		return fmt.Errorf("host: the list of vaults was not changed, code %s", code)
	}
}

// failNote converts the error code of one note into its error, nil when there is none. A refused
// note carries the core's text in detail.
func failNote(code v1.HostVaultErrorCode, detail string) error {
	err := fail(code)
	if err == nil || detail == "" {
		return err
	}
	return fmt.Errorf("%w: %s", err, detail)
}

// readMove converts a move of the wire; the file is at To exactly when To is set.
func readMove(m *v1.MoveResult) domain.MoveResult {
	return domain.MoveResult{
		From:     m.GetFrom(),
		To:       m.GetTo(),
		IsMoved:  m.GetTo() != "",
		Repaired: m.GetRepaired(),
		Dangling: m.GetDangling(),
	}
}

func readVault(v *v1.Vault) domain.Vault {
	return domain.Vault{ID: domain.VaultID(v.GetId()), Name: v.GetName(), Path: v.GetPath()}
}

func (c *Client) RenameNote(ctx context.Context, path, title string) (NoteRename, error) {
	res, err := c.rpc.RenameNote(ctx, request(c, &v1.HostVaultServiceRenameNoteRequest{Path: path, Title: title}))
	if err != nil {
		return NoteRename{}, readError(err)
	}
	if err := failNote(res.Msg.GetError(), res.Msg.GetDetail()); err != nil {
		return NoteRename{}, fmt.Errorf("rename %s: %w", path, err)
	}
	out := NoteRename{
		Path:        res.Msg.GetPath(),
		Title:       res.Msg.GetTitle(),
		IsFilenamed: res.Msg.GetNameSource() == v1.NameSource_NAME_SOURCE_FILENAME,
		Move:        nil,
	}
	if res.Msg.Move != nil {
		move := readMove(res.Msg.GetMove())
		out.Move = &move
	}
	return out, nil
}

func (c *Client) MoveNotes(ctx context.Context, paths []string, folder string) ([]NoteMove, error) {
	res, err := c.rpc.MoveNotes(ctx, request(c, &v1.HostVaultServiceMoveNotesRequest{Paths: paths, Folder: folder}))
	if err != nil {
		return nil, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	out := make([]NoteMove, 0, len(res.Msg.GetMoves()))
	for _, one := range res.Msg.GetMoves() {
		move := NoteMove{Move: readMove(one.GetMove()), Err: failNote(one.GetError(), one.GetDetail())}
		if move.Err != nil {
			move.Move.To, move.Move.IsMoved = "", false
		}
		out = append(out, move)
	}
	return out, nil
}

func (c *Client) RemoveNotes(ctx context.Context, paths []string) ([]NoteRemoval, error) {
	res, err := c.rpc.RemoveNotes(ctx, request(c, &v1.HostVaultServiceRemoveNotesRequest{Paths: paths}))
	if err != nil {
		return nil, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	out := make([]NoteRemoval, 0, len(res.Msg.GetRemovals()))
	for _, one := range res.Msg.GetRemovals() {
		out = append(out, NoteRemoval{
			Path:        one.GetPath(),
			TrashPath:   one.GetTrashPath(),
			BrokenLinks: one.GetBrokenLinks(),
			Err:         failNote(one.GetError(), one.GetDetail()),
		})
	}
	return out, nil
}

func (c *Client) GetProblems(ctx context.Context, checks []string) (VaultProblems, error) {
	res, err := c.rpc.GetVaultProblems(ctx, request(c, &v1.GetVaultProblemsRequest{Checks: checks}))
	if err != nil {
		return VaultProblems{}, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return VaultProblems{}, err
	}
	found := make([]domain.VaultProblem, 0, len(res.Msg.GetProblems()))
	for _, one := range res.Msg.GetProblems() {
		found = append(found, domain.VaultProblem{
			Path:       one.GetPath(),
			Kind:       notesdomain.Check(one.GetCheck()),
			Detail:     one.GetDetail(),
			Target:     domain.Address{Scheme: one.GetTargetScheme(), Value: one.GetTargetValue()},
			Candidates: one.GetCandidates(),
		})
	}
	return VaultProblems{Checks: res.Msg.GetChecks(), Found: found}, nil
}

func (c *Client) GetVaultList(ctx context.Context) ([]VaultEntry, error) {
	res, err := c.rpc.ListHostVaults(ctx, request(c, &v1.ListHostVaultsRequest{}))
	if err != nil {
		return nil, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	out := make([]VaultEntry, 0, len(res.Msg.GetVaults()))
	for _, one := range res.Msg.GetVaults() {
		out = append(out, VaultEntry{
			Vault:     domain.Vault{ID: domain.VaultID(one.GetId()), Name: one.GetName(), Path: one.GetPath()},
			IsMissing: one.GetIsMissing(),
		})
	}
	return out, nil
}

func (c *Client) AddVault(ctx context.Context, root, name string) (domain.Vault, error) {
	res, err := c.rpc.AddVault(ctx, request(c, &v1.HostVaultServiceAddVaultRequest{Root: root, Name: name}))
	if err != nil {
		return domain.Vault{}, readError(err)
	}
	if err := failVaults("add vault "+root, res.Msg.GetError(), res.Msg.GetDetail()); err != nil {
		return domain.Vault{}, err
	}
	return readVault(res.Msg.GetVault()), nil
}

func (c *Client) RenameVault(ctx context.Context, id domain.VaultID, name string) (domain.Vault, error) {
	res, err := c.rpc.RenameVault(ctx, request(c, &v1.HostVaultServiceRenameVaultRequest{Id: string(id), Name: name}))
	if err != nil {
		return domain.Vault{}, readError(err)
	}
	if err := failVaults("rename vault "+string(id), res.Msg.GetError(), res.Msg.GetDetail()); err != nil {
		return domain.Vault{}, err
	}
	return readVault(res.Msg.GetVault()), nil
}

func (c *Client) RemoveVault(ctx context.Context, id domain.VaultID) (string, error) {
	res, err := c.rpc.RemoveVault(ctx, request(c, &v1.HostVaultServiceRemoveVaultRequest{Id: string(id)}))
	if err != nil {
		return "", readError(err)
	}
	if err := failVaults("remove vault "+string(id), res.Msg.GetError(), res.Msg.GetDetail()); err != nil {
		return "", err
	}
	return res.Msg.GetPath(), nil
}

func (c *Client) OpenVault(ctx context.Context, id domain.VaultID) error {
	res, err := c.rpc.OpenVault(ctx, request(c, &v1.HostVaultServiceOpenVaultRequest{Id: string(id)}))
	if err != nil {
		return readError(err)
	}
	return failVaults("open vault "+string(id), res.Msg.GetError(), res.Msg.GetDetail())
}
