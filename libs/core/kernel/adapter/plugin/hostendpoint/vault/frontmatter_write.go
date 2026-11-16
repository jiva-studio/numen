package vault

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// frontmatterWriteLimit is the maximum number of notes one WriteFrontmatter call writes.
const frontmatterWriteLimit = 1000

var errTooManyFrontmatterWrites = fmt.Errorf("too many notes: at most %d per call", frontmatterWriteLimit)

func (e *arrangeCalls) WriteFrontmatter(
	ctx context.Context, req *connect.Request[v1.WriteFrontmatterRequest],
) (*connect.Response[v1.WriteFrontmatterResponse], error) {
	if len(req.Msg.GetNotes()) > frontmatterWriteLimit {
		return nil, connect.NewError(connect.CodeInvalidArgument, errTooManyFrontmatterWrites)
	}
	vault, err := e.begin(req.Header(), manifest.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.WriteFrontmatterResponse{Error: noVault}), nil
	}
	writes := make([]plugin.FrontmatterWrite, len(req.Msg.GetNotes()))
	for i, one := range req.Msg.GetNotes() {
		values, err := readValues(one.GetValues())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("values of %s: %w", one.GetPath(), err))
		}
		writes[i] = plugin.FrontmatterWrite{Path: one.GetPath(), Baseline: readFileStat(one.GetBaseline()), Values: values}
	}
	written, err := vault.Arranger.WriteFrontmatter(ctx, writes)
	if err != nil {
		_, code, callErr := splitNoteError(ctx, err)
		if callErr == nil && code == 0 {
			callErr = connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.WriteFrontmatterResponse{Error: code}), callErr
	}
	out := &v1.WriteFrontmatterResponse{Results: make([]*v1.NoteFrontmatterWritten, len(written))}
	// The results already hold the cancellation of ctx: the notes not written are ErrNotAttempted.
	resultCtx := context.WithoutCancel(ctx)
	for i, one := range written {
		if errors.Is(one.Err, plugin.ErrNotAttempted) {
			out.Results[i] = &v1.NoteFrontmatterWritten{Error: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_ATTEMPTED, Detail: one.Err.Error()}
			continue
		}
		entry, code, callErr := writeLinkEdit(resultCtx, one.Stat, one.Err)
		if callErr != nil || code != 0 {
			return connect.NewResponse(&v1.WriteFrontmatterResponse{Error: code}), callErr
		}
		out.Results[i] = &v1.NoteFrontmatterWritten{Stat: entry.stat, Error: entry.code, Detail: entry.detail}
	}
	return connect.NewResponse(out), nil
}

func readValues(raw []byte) (map[string]any, error) {
	var values map[string]any
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, err
	}
	if values == nil {
		return nil, errors.New("not a JSON object")
	}
	return values, nil
}
