package vault

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"

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
	vault, err := e.begin(req.Header())
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.WriteFrontmatterResponse{Error: noVault}), nil
	}
	writes, err := readFrontmatterWrites(req.Msg.GetNotes())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	written, err := vault.LinkEditor.WriteFrontmatter(ctx, writes)
	if err != nil {
		_, code, callErr := splitNoteError(ctx, err)
		if callErr == nil && code == 0 {
			callErr = connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.WriteFrontmatterResponse{Error: code}), callErr
	}
	results, code, err := writeFrontmatterResults(ctx, written)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.WriteFrontmatterResponse{Error: code}), err
	}
	return connect.NewResponse(&v1.WriteFrontmatterResponse{Results: results}), nil
}

func readFrontmatterWrites(notes []*v1.NoteFrontmatterWrite) ([]plugin.FrontmatterWrite, error) {
	writes := make([]plugin.FrontmatterWrite, len(notes))
	for i, one := range notes {
		values, err := readValues(one.GetValues())
		if err != nil {
			return nil, fmt.Errorf("values of %s: %w", one.GetPath(), err)
		}
		writes[i] = plugin.FrontmatterWrite{Path: one.GetPath(), Baseline: readFileStat(one.GetBaseline()), Values: values}
	}
	return writes, nil
}

func writeFrontmatterResults(ctx context.Context, written []plugin.FrontmatterResult) ([]*v1.NoteFrontmatterWritten, v1.ErrorCode, error) {
	results := make([]*v1.NoteFrontmatterWritten, len(written))
	// The results already hold the cancellation of ctx: the notes not written are ErrNotAttempted.
	resultCtx := context.WithoutCancel(ctx)
	for i, one := range written {
		if errors.Is(one.Err, plugin.ErrNotAttempted) {
			results[i] = &v1.NoteFrontmatterWritten{Error: v1.ErrorCode_ERROR_CODE_NOT_ATTEMPTED, Detail: one.Err.Error()}
			continue
		}
		entry, code, err := writeLinkEdit(resultCtx, one.Stat, one.Err)
		if err != nil || code != 0 {
			return nil, code, err
		}
		results[i] = &v1.NoteFrontmatterWritten{Stat: entry.stat, Error: entry.code, Detail: entry.detail}
	}
	return results, 0, nil
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
