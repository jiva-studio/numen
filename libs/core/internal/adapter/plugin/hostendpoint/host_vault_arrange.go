package hostendpoint

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/wire"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

// batchLimit is the most notes one call moves or removes.
const batchLimit = 50

var errBatchTooLarge = fmt.Errorf("a call names at most %d notes", batchLimit)

// getNoteFailure is the error code of a note that was not changed, and false for an error that is
// not about the note.
func getNoteFailure(err error) (v1.ErrorCode, bool) {
	if code, ok := wire.GetErrorCode(err); ok {
		return code, true
	}
	if errors.Is(err, fs.ErrNotExist) {
		return v1.ErrorCode_ERROR_CODE_MISSING, true
	}
	return v1.ErrorCode_ERROR_CODE_UNSPECIFIED, false
}

func writeMove(m domain.MoveResult) *v1.MoveResult {
	if !m.IsMoved {
		return &v1.MoveResult{From: m.From}
	}
	return &v1.MoveResult{From: m.From, To: m.To, Repaired: m.Repaired, Dangling: m.Dangling}
}

func checkBatch(paths []string) error {
	if len(paths) > batchLimit {
		return connect.NewError(connect.CodeInvalidArgument, errBatchTooLarge)
	}
	return nil
}

func (e *vaultEndpoint) GetVaultProblems(
	ctx context.Context, req *connect.Request[v1.GetVaultProblemsRequest],
) (*connect.Response[v1.GetVaultProblemsResponse], error) {
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.GetVaultProblemsResponse{Error: noVault}), nil
	}
	named := make([]domain.Check, 0, len(req.Msg.GetChecks()))
	for _, one := range req.Msg.GetChecks() {
		named = append(named, domain.Check(one))
	}
	ran, found, err := vault.GetProblems(ctx, named)
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.GetVaultProblemsResponse{Error: code}), err
	}
	out := &v1.GetVaultProblemsResponse{
		Ran:      make([]string, 0, len(ran)),
		Problems: make([]*v1.HostVaultProblem, 0, len(found)),
	}
	for _, one := range ran {
		out.Ran = append(out.Ran, string(one))
	}
	for _, one := range found {
		out.Problems = append(out.Problems, &v1.HostVaultProblem{
			Path: one.Path, Check: string(one.Kind), Detail: one.Detail,
			TargetScheme: one.Target.Scheme, TargetValue: one.Target.Value, Candidates: one.Candidates,
		})
	}
	return connect.NewResponse(out), nil
}

func (e *vaultEndpoint) RenameNote(
	ctx context.Context, req *connect.Request[v1.HostVaultServiceRenameNoteRequest],
) (*connect.Response[v1.HostVaultServiceRenameNoteResponse], error) {
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.HostVaultServiceRenameNoteResponse{Error: noVault}), nil
	}
	to, isFilenamed, moved, err := vault.RenameNote(ctx, req.Msg.GetPath(), req.Msg.GetTitle())
	// A note whose file landed under the new name is renamed, whatever came apart after it.
	if err != nil && moved == nil {
		if failure, ok := getNoteFailure(err); ok {
			return connect.NewResponse(&v1.HostVaultServiceRenameNoteResponse{Path: req.Msg.GetPath(), Failure: &failure}), nil
		}
		code, err := answer(ctx, err)
		return connect.NewResponse(&v1.HostVaultServiceRenameNoteResponse{Error: code}), err
	}
	out := &v1.HostVaultServiceRenameNoteResponse{
		Path: to, Title: strings.TrimSpace(req.Msg.GetTitle()), NameSource: v1.NameSource_NAME_SOURCE_FRONTMATTER,
	}
	if isFilenamed {
		out.NameSource = v1.NameSource_NAME_SOURCE_FILENAME
	}
	if moved != nil {
		out.Moved = writeMove(*moved)
	}
	return connect.NewResponse(out), nil
}

func (e *vaultEndpoint) MoveNotes(
	ctx context.Context, req *connect.Request[v1.HostVaultServiceMoveNotesRequest],
) (*connect.Response[v1.HostVaultServiceMoveNotesResponse], error) {
	if err := checkBatch(req.Msg.GetPaths()); err != nil {
		return nil, err
	}
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.HostVaultServiceMoveNotesResponse{Error: noVault}), nil
	}
	out := &v1.HostVaultServiceMoveNotesResponse{Moves: make([]*v1.HostNoteMove, 0, len(req.Msg.GetPaths()))}
	for _, from := range req.Msg.GetPaths() {
		if err := ctx.Err(); err != nil {
			return nil, connect.NewError(connect.CodeCanceled, err)
		}
		moved, err := vault.MoveNote(ctx, from, path.Join(req.Msg.GetFolder(), path.Base(from)))
		entry := &v1.HostNoteMove{Moved: writeMove(moved)}
		if err != nil && !moved.IsMoved {
			failure, ok := getNoteFailure(err)
			if !ok {
				code, err := answer(ctx, err)
				return connect.NewResponse(&v1.HostVaultServiceMoveNotesResponse{Error: code}), err
			}
			entry = &v1.HostNoteMove{Moved: &v1.MoveResult{From: from}, Failure: &failure}
		}
		out.Moves = append(out.Moves, entry)
	}
	return connect.NewResponse(out), nil
}

func (e *vaultEndpoint) RemoveNotes(
	ctx context.Context, req *connect.Request[v1.HostVaultServiceRemoveNotesRequest],
) (*connect.Response[v1.HostVaultServiceRemoveNotesResponse], error) {
	if err := checkBatch(req.Msg.GetPaths()); err != nil {
		return nil, err
	}
	_, vault, err := e.begin(req.Header(), pluginport.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.HostVaultServiceRemoveNotesResponse{Error: noVault}), nil
	}
	out := &v1.HostVaultServiceRemoveNotesResponse{Removals: make([]*v1.HostNoteRemoval, 0, len(req.Msg.GetPaths()))}
	for _, target := range req.Msg.GetPaths() {
		if err := ctx.Err(); err != nil {
			return nil, connect.NewError(connect.CodeCanceled, err)
		}
		trashed, dangling, err := vault.RemoveNote(ctx, target)
		entry := &v1.HostNoteRemoval{Path: target, Trashed: trashed, Dangling: dangling}
		if err != nil && trashed == "" {
			failure, ok := getNoteFailure(err)
			if !ok {
				code, err := answer(ctx, err)
				return connect.NewResponse(&v1.HostVaultServiceRemoveNotesResponse{Error: code}), err
			}
			entry = &v1.HostNoteRemoval{Path: target, Failure: &failure}
		}
		out.Removals = append(out.Removals, entry)
	}
	return connect.NewResponse(out), nil
}
