package hostendpoint

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// textEndpoint answers HostTextService for the plugins that call the host.
type textEndpoint struct {
	endpoint *Endpoint
}

var _ numenv1connect.HostTextServiceHandler = (*textEndpoint)(nil)

// textCall is an authorized text call: the plugin that made it, the text versions, and the vault
// it works. Texts is nil where the host has no text versions or no vault to offer.
type textCall struct {
	pluginID string
	texts    TextVersions
	vault    domain.Vault
}

func (e *textEndpoint) begin(header http.Header, permission string) (textCall, error) {
	pluginID, err := e.endpoint.Authorize(header.Get("Authorization"), permission)
	if err != nil {
		return textCall{}, err
	}
	call := textCall{pluginID: pluginID}
	texts := e.endpoint.callers.Lent.GetPluginTexts()
	if texts == nil {
		return call, nil
	}
	vault, err := e.endpoint.getVault(header)
	if err != nil || vault == nil {
		return call, err
	}
	v, ok := vault.Switch.GetVault()
	if !ok {
		return call, nil
	}
	call.texts, call.vault = texts, v
	return call, nil
}

func answerText(ctx context.Context, err error) (v1.TextErrorCode, error) {
	if err == nil {
		return v1.TextErrorCode_TEXT_ERROR_CODE_UNSPECIFIED, nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return 0, connect.NewError(connect.CodeCanceled, ctxErr)
	}
	switch {
	case errors.Is(err, text.ErrNoText):
		return v1.TextErrorCode_TEXT_ERROR_CODE_MISSING, nil
	case errors.Is(err, text.ErrInvalidVersion):
		return v1.TextErrorCode_TEXT_ERROR_CODE_INVALID, nil
	case errors.Is(err, text.ErrNotTextPlugin):
		return v1.TextErrorCode_TEXT_ERROR_CODE_NOT_OWNER, nil
	}
	return 0, connect.NewError(connect.CodeInternal, err)
}

const noTexts = v1.TextErrorCode_TEXT_ERROR_CODE_NO_VAULT

func (e *textEndpoint) PutTextVersion(
	ctx context.Context, req *connect.Request[v1.PutTextVersionRequest],
) (*connect.Response[v1.PutTextVersionResponse], error) {
	call, err := e.begin(req.Header(), manifest.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if call.texts == nil {
		return connect.NewResponse(&v1.PutTextVersionResponse{Error: noTexts}), nil
	}
	m := req.Msg
	version, err := call.texts.PutVersion(ctx, call.vault, call.pluginID, plugin.TextPut{
		Path: m.GetPath(), Key: m.GetKey(), Source: domain.TextVersionID(m.GetSourceVersionId()),
		Kind: domain.TextKind(m.GetKind()), InputHash: m.GetInputHash(), File: m.GetFile(),
		SourceProperty: domain.TextSourceProperty{Key: m.GetSourceProperty().GetKey(), Value: m.GetSourceProperty().GetValue()},
	})
	code, err := answerText(ctx, err)
	if err != nil {
		return nil, err
	}
	res := &v1.PutTextVersionResponse{Error: code}
	if code == v1.TextErrorCode_TEXT_ERROR_CODE_UNSPECIFIED {
		res.Version = writeVersion(version)
	}
	return connect.NewResponse(res), nil
}

func (e *textEndpoint) ListTextVersions(
	ctx context.Context, req *connect.Request[v1.ListTextVersionsRequest],
) (*connect.Response[v1.ListTextVersionsResponse], error) {
	call, err := e.begin(req.Header(), manifest.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if call.texts == nil {
		return connect.NewResponse(&v1.ListTextVersionsResponse{Error: noTexts}), nil
	}
	listing, err := call.texts.ListVersions(ctx, call.vault, req.Msg.GetPath())
	code, err := answerText(ctx, err)
	if err != nil {
		return nil, err
	}
	res := &v1.ListTextVersionsResponse{Error: code, LiveVersionId: string(listing.Live)}
	for _, one := range listing.Versions {
		res.Versions = append(res.Versions, writeVersion(one))
	}
	return connect.NewResponse(res), nil
}

func (e *textEndpoint) ReadText(
	ctx context.Context, req *connect.Request[v1.ReadTextRequest],
) (*connect.Response[v1.ReadTextResponse], error) {
	call, err := e.begin(req.Header(), manifest.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if call.texts == nil {
		return connect.NewResponse(&v1.ReadTextResponse{Error: noTexts}), nil
	}
	if req.Msg.Length != nil {
		return serveTextRange(ctx, call, req.Msg)
	}
	page, err := call.texts.ReadText(ctx, call.vault, req.Msg.GetPath(), domain.TextVersionID(req.Msg.GetVersionId()), req.Msg.GetCursor())
	code, err := answerText(ctx, err)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.ReadTextResponse{
		Error: code, Segments: writeSegments(page.Segments), NextCursor: page.NextCursor,
	}), nil
}

func serveTextRange(ctx context.Context, call textCall, req *v1.ReadTextRequest) (*connect.Response[v1.ReadTextResponse], error) {
	if req.GetStart() < 0 || req.GetLength() < 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("invalid range of the text of %s: start %d and length %d must be >= 0", req.GetPath(), req.GetStart(), req.GetLength()))
	}
	read, err := call.texts.ReadTextRange(ctx, call.vault, req.GetPath(), domain.TextVersionID(req.GetVersionId()),
		int(req.GetStart()), int(req.GetLength()))
	code, err := answerText(ctx, err)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.ReadTextResponse{
		Error: code, Segments: writeSegments(read.Segments), Start: int64(read.Start), Size: int64(read.Size),
	}), nil
}
