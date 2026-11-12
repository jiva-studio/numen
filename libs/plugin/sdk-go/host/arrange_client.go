package host

import (
	"context"
	"fmt"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	notesdomain "github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

var _ VaultList = (*Client)(nil)

func getVaultsError(code v1.VaultsErrorCode, detail string) error {
	return addDetail(readVaultsError(code), detail)
}

func addDetail(err error, detail string) error {
	if err == nil || detail == "" {
		return err
	}
	return notesdomain.NewDetailError(detail, err)
}

func readVaultsError(code v1.VaultsErrorCode) error {
	switch code {
	case v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNSPECIFIED:
		return nil
	case v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNREADABLE:
		return ErrVaultUnreadable
	case v1.VaultsErrorCode_VAULTS_ERROR_CODE_COPY:
		return ErrVaultDuplicate
	case v1.VaultsErrorCode_VAULTS_ERROR_CODE_OVERLAPS:
		return ErrVaultOverlaps
	case v1.VaultsErrorCode_VAULTS_ERROR_CODE_NAME_TAKEN:
		return ErrVaultNameTaken
	case v1.VaultsErrorCode_VAULTS_ERROR_CODE_LAST_VAULT:
		return ErrLastVault
	case v1.VaultsErrorCode_VAULTS_ERROR_CODE_CURRENT:
		return ErrVaultActive
	case v1.VaultsErrorCode_VAULTS_ERROR_CODE_NOT_FOUND:
		return ErrUnknownVault
	case v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNSAVED:
		return ErrUnsavedChanges
	default:
		return fmt.Errorf("the list of vaults is unchanged; the host returned the code %s", code)
	}
}

func getNoteError(code v1.HostVaultErrorCode, detail string) error {
	return addDetail(fail(code), detail)
}

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
	if err := getNoteError(res.Msg.GetError(), res.Msg.GetDetail()); err != nil {
		return NoteRename{}, err
	}
	out := NoteRename{
		Path:        res.Msg.GetPath(),
		Title:       res.Msg.GetTitle(),
		IsFilenamed: res.Msg.GetNameSource() == v1.NameSource_NAME_SOURCE_FILENAME,
		Move:        nil,
	}
	if res.Msg.GetMove() != nil {
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
		move := NoteMove{Move: readMove(one.GetMove()), Err: getNoteError(one.GetError(), one.GetDetail())}
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
			Err:         getNoteError(one.GetError(), one.GetDetail()),
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
	if err := getVaultsError(res.Msg.GetError(), res.Msg.GetDetail()); err != nil {
		return domain.Vault{}, err
	}
	return readVault(res.Msg.GetVault()), nil
}

func (c *Client) RenameVault(ctx context.Context, id domain.VaultID, name string) (domain.Vault, error) {
	res, err := c.rpc.RenameVault(ctx, request(c, &v1.HostVaultServiceRenameVaultRequest{Id: string(id), Name: name}))
	if err != nil {
		return domain.Vault{}, readError(err)
	}
	if err := getVaultsError(res.Msg.GetError(), res.Msg.GetDetail()); err != nil {
		return domain.Vault{}, err
	}
	return readVault(res.Msg.GetVault()), nil
}

func (c *Client) RemoveVault(ctx context.Context, id domain.VaultID) (string, error) {
	res, err := c.rpc.RemoveVault(ctx, request(c, &v1.HostVaultServiceRemoveVaultRequest{Id: string(id)}))
	if err != nil {
		return "", readError(err)
	}
	if err := getVaultsError(res.Msg.GetError(), res.Msg.GetDetail()); err != nil {
		return "", err
	}
	return res.Msg.GetPath(), nil
}

func (c *Client) OpenVault(ctx context.Context, id domain.VaultID) error {
	res, err := c.rpc.OpenVault(ctx, request(c, &v1.HostVaultServiceOpenVaultRequest{Id: string(id)}))
	if err != nil {
		return readError(err)
	}
	return getVaultsError(res.Msg.GetError(), res.Msg.GetDetail())
}

func (c *Client) AddLinks(ctx context.Context, additions []LinkAddition) ([]LinkAdded, error) {
	links := make([]*v1.HostLinkAddition, len(additions))
	for i, one := range additions {
		links[i] = &v1.HostLinkAddition{
			From: one.From, Baseline: writeFileStat(one.Baseline), To: one.Link.Target.GetWritten(),
			Role: writeRole(one.Link.Role), Type: one.Link.Type, Label: one.Link.Label, Note: one.Link.Why,
		}
	}
	res, err := c.rpc.AddLinks(ctx, request(c, &v1.HostVaultServiceAddLinksRequest{Links: links}))
	if err != nil {
		return nil, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	out := make([]LinkAdded, 0, len(res.Msg.GetResults()))
	for _, one := range res.Msg.GetResults() {
		added := LinkAdded{Err: getNoteError(one.GetError(), one.GetDetail())}
		if added.Err == nil {
			added.Stat = readFileStat(one.GetStat())
		}
		out = append(out, added)
	}
	return out, nil
}

func (c *Client) UpdateLinks(
	ctx context.Context, from string, to domain.Address, change domain.Link, baseline domain.FileStat,
) (domain.FileStat, error) {
	if err := checkRole(change.Role); err != nil {
		return domain.FileStat{}, err
	}
	res, err := c.rpc.UpdateLinks(ctx, request(c, &v1.HostVaultServiceUpdateLinksRequest{
		From: from, To: to.GetWritten(), Baseline: writeFileStat(baseline),
		Role: writeRole(change.Role), Type: change.Type, Label: change.Label, Note: change.Why,
	}))
	if err != nil {
		return domain.FileStat{}, readError(err)
	}
	if err := getNoteError(res.Msg.GetError(), res.Msg.GetDetail()); err != nil {
		return domain.FileStat{}, err
	}
	return readFileStat(res.Msg.GetStat()), nil
}

func (c *Client) RemoveLinks(
	ctx context.Context, from string, to domain.Address, role domain.LinkRole, baseline domain.FileStat,
) (domain.FileStat, error) {
	if err := checkRole(role); err != nil {
		return domain.FileStat{}, err
	}
	res, err := c.rpc.RemoveLinks(ctx, request(c, &v1.HostVaultServiceRemoveLinksRequest{
		From: from, To: to.GetWritten(), Baseline: writeFileStat(baseline), Role: writeRole(role),
	}))
	if err != nil {
		return domain.FileStat{}, readError(err)
	}
	if err := getNoteError(res.Msg.GetError(), res.Msg.GetDetail()); err != nil {
		return domain.FileStat{}, err
	}
	return readFileStat(res.Msg.GetStat()), nil
}

func checkRole(role domain.LinkRole) error {
	if role != "" && writeRole(role) == v1.Role_ROLE_UNSPECIFIED {
		return fmt.Errorf("%q is not a role a link can have", role)
	}
	return nil
}
