package vault

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// batchLimit is the maximum number of notes one call moves or removes.
const batchLimit = 50

var errBatchTooLarge = fmt.Errorf("too many notes: at most %d per call", batchLimit)

func splitNoteError(ctx context.Context, err error) (noteCode, callCode v1.HostVaultErrorCode, callErr error) {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return 0, 0, connect.NewError(connect.CodeCanceled, ctxErr)
	}
	if errors.Is(err, plugin.ErrNoVault) {
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

func (e *arrangeCalls) GetVaultProblems(
	ctx context.Context, req *connect.Request[v1.GetVaultProblemsRequest],
) (*connect.Response[v1.GetVaultProblemsResponse], error) {
	_, vault, err := e.begin(req.Header(), manifest.PermissionVaultRead)
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
	checkNames, problems, err := vault.Arranger.GetProblems(ctx, checks)
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

func (e *arrangeCalls) RenameNote(
	ctx context.Context, req *connect.Request[v1.HostVaultServiceRenameNoteRequest],
) (*connect.Response[v1.HostVaultServiceRenameNoteResponse], error) {
	_, vault, err := e.begin(req.Header(), manifest.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.HostVaultServiceRenameNoteResponse{Error: noVault}), nil
	}
	newPath, isFilenamed, move, err := vault.Arranger.RenameNote(ctx, req.Msg.GetPath(), req.Msg.GetTitle())
	// A note whose file was moved to the new name is renamed, even if a later step failed.
	if err != nil && move == nil {
		noteCode, code, callErr := splitNoteError(ctx, err)
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

func (e *arrangeCalls) MoveNotes(
	ctx context.Context, req *connect.Request[v1.HostVaultServiceMoveNotesRequest],
) (*connect.Response[v1.HostVaultServiceMoveNotesResponse], error) {
	if err := checkBatch(req.Msg.GetPaths()); err != nil {
		return nil, err
	}
	_, vault, err := e.begin(req.Header(), manifest.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.HostVaultServiceMoveNotesResponse{Error: noVault}), nil
	}
	out := &v1.HostVaultServiceMoveNotesResponse{Moves: make([]*v1.HostNoteMove, 0, len(req.Msg.GetPaths()))}
	for _, from := range req.Msg.GetPaths() {
		entry, code, err := moveNote(ctx, vault.Arranger, from, req.Msg.GetFolder())
		if err != nil || code != 0 {
			return connect.NewResponse(&v1.HostVaultServiceMoveNotesResponse{Error: code}), err
		}
		out.Moves = append(out.Moves, entry)
	}
	return connect.NewResponse(out), nil
}

func moveNote(ctx context.Context, vault Arranger, from, folder string) (*v1.HostNoteMove, v1.HostVaultErrorCode, error) {
	notAttempted := &v1.HostNoteMove{
		Move: &v1.MoveResult{From: from}, Error: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_ATTEMPTED,
		Detail: fmt.Sprintf("the call was canceled before %s was moved", from),
	}
	if isCanceled(ctx) {
		return notAttempted, 0, nil
	}
	move, err := vault.MoveNote(ctx, from, path.Join(folder, path.Base(from)))
	if err == nil || move.IsMoved {
		return &v1.HostNoteMove{Move: writeMove(move)}, 0, nil
	}
	if isCanceled(ctx) {
		return notAttempted, 0, nil
	}
	noteCode, code, callErr := splitNoteError(ctx, err)
	if callErr != nil || code != 0 {
		return nil, code, callErr
	}
	return &v1.HostNoteMove{Move: &v1.MoveResult{From: from}, Error: noteCode, Detail: err.Error()}, 0, nil
}

func (e *arrangeCalls) RemoveNotes(
	ctx context.Context, req *connect.Request[v1.HostVaultServiceRemoveNotesRequest],
) (*connect.Response[v1.HostVaultServiceRemoveNotesResponse], error) {
	if err := checkBatch(req.Msg.GetPaths()); err != nil {
		return nil, err
	}
	_, vault, err := e.begin(req.Header(), manifest.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.HostVaultServiceRemoveNotesResponse{Error: noVault}), nil
	}
	out := &v1.HostVaultServiceRemoveNotesResponse{Removals: make([]*v1.HostNoteRemoval, 0, len(req.Msg.GetPaths()))}
	for _, target := range req.Msg.GetPaths() {
		entry, code, err := removeNote(ctx, vault.Arranger, target)
		if err != nil || code != 0 {
			return connect.NewResponse(&v1.HostVaultServiceRemoveNotesResponse{Error: code}), err
		}
		out.Removals = append(out.Removals, entry)
	}
	return connect.NewResponse(out), nil
}

func removeNote(ctx context.Context, vault Arranger, target string) (*v1.HostNoteRemoval, v1.HostVaultErrorCode, error) {
	notAttempted := &v1.HostNoteRemoval{
		Path: target, Error: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_ATTEMPTED,
		Detail: fmt.Sprintf("the call was canceled before %s was removed", target),
	}
	if isCanceled(ctx) {
		return notAttempted, 0, nil
	}
	trashPath, brokenLinks, err := vault.RemoveNote(ctx, target)
	if err == nil || trashPath != "" {
		return &v1.HostNoteRemoval{Path: target, TrashPath: trashPath, BrokenLinks: brokenLinks}, 0, nil
	}
	if isCanceled(ctx) {
		return notAttempted, 0, nil
	}
	noteCode, code, callErr := splitNoteError(ctx, err)
	if callErr != nil || code != 0 {
		return nil, code, callErr
	}
	return &v1.HostNoteRemoval{Path: target, Error: noteCode, Detail: err.Error()}, 0, nil
}

func isCanceled(ctx context.Context) bool {
	return ctx.Err() != nil
}
func (e *arrangeCalls) GetParseProblems(
	ctx context.Context, req *connect.Request[v1.GetParseProblemsRequest],
) (*connect.Response[v1.GetParseProblemsResponse], error) {
	_, vault, err := e.begin(req.Header(), manifest.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.GetParseProblemsResponse{Error: noVault}), nil
	}
	target, err := vault.Arranger.GetParseProblems(ctx)
	code, err := answer(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.GetParseProblemsResponse{Error: code}), err
	}
	problems := make([]*v1.HostParseProblem, 0, len(target))
	for _, one := range target {
		problems = append(problems, &v1.HostParseProblem{Path: one.Path, Detail: one.Detail})
	}
	return connect.NewResponse(&v1.GetParseProblemsResponse{Problems: problems}), nil
}
