package hostendpoint

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugindir"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/textwire"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/textport"
)

// textEndpoint answers HostTextService for the plugins that call the host.
type textEndpoint struct {
	endpoint *Endpoint
}

var _ numenv1connect.HostTextServiceHandler = (*textEndpoint)(nil)

// begin authorizes a call and returns the plugin that made it and the text
// versions of the vault the host shows, which are nil while it shows none.
func (e *textEndpoint) begin(header http.Header, permission string) (string, pluginport.PluginTexts, error) {
	pluginID, err := e.endpoint.authorize(header.Get("Authorization"), permission)
	if err != nil {
		return "", nil, err
	}
	return pluginID, e.endpoint.callers.GetPluginTexts(), nil
}

// answerText sorts the failure of a text call: an answer the protocol names is
// a code, the end of the caller's context is cancelled, and anything else is internal.
func answerText(ctx context.Context, err error) (v1.TextErrorCode, error) {
	if err == nil {
		return v1.TextErrorCode_TEXT_ERROR_CODE_UNSPECIFIED, nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return 0, connect.NewError(connect.CodeCanceled, ctxErr)
	}
	switch {
	case errors.Is(err, textport.ErrNoText):
		return v1.TextErrorCode_TEXT_ERROR_CODE_MISSING, nil
	case errors.Is(err, textport.ErrInvalidVersion):
		return v1.TextErrorCode_TEXT_ERROR_CODE_INVALID, nil
	case errors.Is(err, textport.ErrNotTextPlugin):
		return v1.TextErrorCode_TEXT_ERROR_CODE_NOT_OWNER, nil
	}
	return 0, connect.NewError(connect.CodeInternal, err)
}

const noTexts = v1.TextErrorCode_TEXT_ERROR_CODE_NO_VAULT

func (e *textEndpoint) PutTextVersion(
	ctx context.Context, req *connect.Request[v1.PutTextVersionRequest],
) (*connect.Response[v1.PutTextVersionResponse], error) {
	pluginID, texts, err := e.begin(req.Header(), plugindir.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if texts == nil {
		return connect.NewResponse(&v1.PutTextVersionResponse{Error: noTexts}), nil
	}
	m := req.Msg
	version, err := texts.PutVersion(ctx, pluginID, pluginport.PluginTextPut{
		Path: m.GetPath(), Key: m.GetKey(), Input: domain.TextVersionID(m.GetInputVersionId()),
		Format: textwire.ReadFormat(m.GetFormat()), InputHash: m.GetInputHash(), Hash: m.GetHash(),
	})
	code, err := answerText(ctx, err)
	if err != nil {
		return nil, err
	}
	res := &v1.PutTextVersionResponse{Error: code}
	if code == v1.TextErrorCode_TEXT_ERROR_CODE_UNSPECIFIED {
		res.Version = textwire.WriteVersion(version)
	}
	return connect.NewResponse(res), nil
}

func (e *textEndpoint) RemoveTextVersion(
	ctx context.Context, req *connect.Request[v1.RemoveTextVersionRequest],
) (*connect.Response[v1.RemoveTextVersionResponse], error) {
	pluginID, texts, err := e.begin(req.Header(), plugindir.PermissionVaultWrite)
	if err != nil {
		return nil, err
	}
	if texts == nil {
		return connect.NewResponse(&v1.RemoveTextVersionResponse{Error: noTexts}), nil
	}
	code, err := answerText(ctx, texts.RemoveVersion(ctx, pluginID, req.Msg.GetPath(), req.Msg.GetKey()))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.RemoveTextVersionResponse{Error: code}), nil
}

func (e *textEndpoint) ListTextVersions(
	ctx context.Context, req *connect.Request[v1.ListTextVersionsRequest],
) (*connect.Response[v1.ListTextVersionsResponse], error) {
	_, texts, err := e.begin(req.Header(), plugindir.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if texts == nil {
		return connect.NewResponse(&v1.ListTextVersionsResponse{Error: noTexts}), nil
	}
	listing, err := texts.ListVersions(ctx, req.Msg.GetPath())
	code, err := answerText(ctx, err)
	if err != nil {
		return nil, err
	}
	res := &v1.ListTextVersionsResponse{Error: code, LiveVersionId: string(listing.Live)}
	for _, one := range listing.Versions {
		res.Versions = append(res.Versions, textwire.WriteVersion(one))
	}
	return connect.NewResponse(res), nil
}

func (e *textEndpoint) ReadText(
	ctx context.Context, req *connect.Request[v1.ReadTextRequest],
) (*connect.Response[v1.ReadTextResponse], error) {
	_, texts, err := e.begin(req.Header(), plugindir.PermissionVaultRead)
	if err != nil {
		return nil, err
	}
	if texts == nil {
		return connect.NewResponse(&v1.ReadTextResponse{Error: noTexts}), nil
	}
	page, err := texts.ReadText(ctx, req.Msg.GetPath(), domain.TextVersionID(req.Msg.GetVersionId()), req.Msg.GetCursor())
	code, err := answerText(ctx, err)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.ReadTextResponse{
		Error: code, Parts: textwire.WriteParts(page.Parts), NextCursor: page.NextCursor,
	}), nil
}
