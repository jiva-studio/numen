package hostendpoint

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/hostvault"
	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/wire"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
	vaultport "github.com/jiva-studio/numen/libs/core/usecase/vault/vaultcontract"
)

// batchLimit is the most notes one call moves or removes.
const batchLimit = 50

var errBatchTooLarge = fmt.Errorf("a call names at most %d notes", batchLimit)

// getNoteErrorCode is the error code of a note that was not changed, and false for an error that
// has no code.
func getNoteErrorCode(err error) (v1.HostVaultErrorCode, bool) {
	if errors.Is(err, vaultport.ErrOutside) {
		return v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_OUTSIDE_VAULT, true
	}
	if code, ok := wire.GetErrorCode(err); ok {
		return convertNoteErrorCode(code), true
	}
	return getVaultErrorCode(err)
}

// convertNoteErrorCode is the HostVaultErrorCode of a per-note error code.
func convertNoteErrorCode(code v1.ErrorCode) v1.HostVaultErrorCode {
	switch code {
	case v1.ErrorCode_ERROR_CODE_NOT_FOUND:
		return v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_FOUND
	case v1.ErrorCode_ERROR_CODE_NOT_A_NOTE:
		return v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_A_NOTE
	case v1.ErrorCode_ERROR_CODE_NOT_TEXT:
		return v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_TEXT
	case v1.ErrorCode_ERROR_CODE_TOO_LARGE:
		return v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_TOO_LARGE
	case v1.ErrorCode_ERROR_CODE_BODY_UNWRITABLE:
		return v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_BODY_UNWRITABLE
	case v1.ErrorCode_ERROR_CODE_UNREADABLE:
		return v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_UNREADABLE
	case v1.ErrorCode_ERROR_CODE_ALREADY_EXISTS:
		return v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_ALREADY_EXISTS
	case v1.ErrorCode_ERROR_CODE_INVALID_NAME:
		return v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_INVALID_NAME
	case v1.ErrorCode_ERROR_CODE_CONFLICT:
		return v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_CONFLICT
	case v1.ErrorCode_ERROR_CODE_UNSPECIFIED:
	}
	return v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_UNSPECIFIED
}

// answerNote splits the error of a change of one note into the error code of that note and the
// error code of the call: a vault the host does not show answers NO_VAULT for the call, a cancelled
// call fails the call, an error with no code is the note's INTERNAL, and every other error is the
// note's own code.
func answerNote(ctx context.Context, err error) (noteCode, callCode v1.HostVaultErrorCode, callErr error) {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return 0, 0, connect.NewError(connect.CodeCanceled, ctxErr)
	}
	if errors.Is(err, hostvault.ErrNoVault) {
		return 0, noVault, nil
	}
	code, ok := getNoteErrorCode(err)
	if !ok {
		return v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_INTERNAL, 0, nil
	}
	return code, 0, nil
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
	checks := make([]domain.Check, 0, len(req.Msg.GetChecks()))
	for _, one := range req.Msg.GetChecks() {
		checks = append(checks, domain.Check(one))
	}
	checkNames, problems, err := vault.GetProblems(ctx, checks)
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.GetVaultProblemsResponse{Error: code}), err
	}
	out := &v1.GetVaultProblemsResponse{
		Checks:   make([]string, 0, len(checkNames)),
		Problems: make([]*v1.HostVaultProblem, 0, len(problems)),
	}
	for _, one := range checkNames {
		out.Checks = append(out.Checks, string(one))
	}
	for _, one := range problems {
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
	newPath, isFilenamed, move, err := vault.RenameNote(ctx, req.Msg.GetPath(), req.Msg.GetTitle())
	// A note whose file landed under the new name is renamed, whatever came apart after it.
	if err != nil && move == nil {
		noteCode, code, callErr := answerNote(ctx, err)
		if callErr != nil || code != 0 {
			return connect.NewResponse(&v1.HostVaultServiceRenameNoteResponse{Error: code}), callErr
		}
		return connect.NewResponse(&v1.HostVaultServiceRenameNoteResponse{
			Path: req.Msg.GetPath(), Error: noteCode, Detail: err.Error(),
		}), nil
	}
	out := &v1.HostVaultServiceRenameNoteResponse{
		Path: newPath, Title: strings.TrimSpace(req.Msg.GetTitle()), NameSource: v1.NameSource_NAME_SOURCE_FRONTMATTER,
	}
	if isFilenamed {
		out.NameSource = v1.NameSource_NAME_SOURCE_FILENAME
	}
	if move != nil {
		out.Move = writeMove(*move)
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
		entry, code, err := moveNote(ctx, vault, from, req.Msg.GetFolder())
		if err != nil || code != 0 {
			return connect.NewResponse(&v1.HostVaultServiceMoveNotesResponse{Error: code}), err
		}
		out.Moves = append(out.Moves, entry)
	}
	return connect.NewResponse(out), nil
}

// moveNote moves one note of a batch into the folder and returns its entry, or the error of the
// call. A note the call was cancelled before is answered NOT_ATTEMPTED.
func moveNote(ctx context.Context, vault pluginport.PluginVault, from, folder string) (*v1.HostNoteMove, v1.HostVaultErrorCode, error) {
	notAttempted := &v1.HostNoteMove{
		Move: &v1.MoveResult{From: from}, Error: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_ATTEMPTED,
		Detail: fmt.Sprintf("the call was cancelled before %s was moved", from),
	}
	if isCancelled(ctx) {
		return notAttempted, 0, nil
	}
	move, err := vault.MoveNote(ctx, from, path.Join(folder, path.Base(from)))
	if err == nil || move.IsMoved {
		return &v1.HostNoteMove{Move: writeMove(move)}, 0, nil
	}
	if isCancelled(ctx) {
		return notAttempted, 0, nil
	}
	noteCode, code, callErr := answerNote(ctx, err)
	if callErr != nil || code != 0 {
		return nil, code, callErr
	}
	return &v1.HostNoteMove{Move: &v1.MoveResult{From: from}, Error: noteCode, Detail: err.Error()}, 0, nil
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
		entry, code, err := removeNote(ctx, vault, target)
		if err != nil || code != 0 {
			return connect.NewResponse(&v1.HostVaultServiceRemoveNotesResponse{Error: code}), err
		}
		out.Removals = append(out.Removals, entry)
	}
	return connect.NewResponse(out), nil
}

// removeNote moves one note of a batch to the trash and returns its entry, or the error of the
// call. A note the call was cancelled before is answered NOT_ATTEMPTED.
func removeNote(ctx context.Context, vault pluginport.PluginVault, target string) (*v1.HostNoteRemoval, v1.HostVaultErrorCode, error) {
	notAttempted := &v1.HostNoteRemoval{
		Path: target, Error: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_ATTEMPTED,
		Detail: fmt.Sprintf("the call was cancelled before %s was removed", target),
	}
	if isCancelled(ctx) {
		return notAttempted, 0, nil
	}
	trashPath, brokenLinks, err := vault.RemoveNote(ctx, target)
	if err == nil || trashPath != "" {
		return &v1.HostNoteRemoval{Path: target, TrashPath: trashPath, BrokenLinks: brokenLinks}, 0, nil
	}
	if isCancelled(ctx) {
		return notAttempted, 0, nil
	}
	noteCode, code, callErr := answerNote(ctx, err)
	if callErr != nil || code != 0 {
		return nil, code, callErr
	}
	return &v1.HostNoteRemoval{Path: target, Error: noteCode, Detail: err.Error()}, 0, nil
}

// isCancelled reports whether the call was cancelled, after which a batch changes no further note.
func isCancelled(ctx context.Context) bool {
	return ctx.Err() != nil
}
