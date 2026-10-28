// Package texts serves a window's TextService: the live text of a file of the
// shown vault, and a person's edit of it, for the views of plugins.
package texts

import (
	"context"
	"errors"
	"io/fs"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/usecase/textversion"
)

// FileText is the live text of a file and a person's edits of it.
type FileText interface {
	ReadLiveText(ctx context.Context, v domain.Vault, path string) (domain.VersionText, bool, error)
	CreateEdit(ctx context.Context, v domain.Vault, edit textversion.TextEdit) (domain.TextVersion, error)
}

// VaultProvider names the vault the window shows. A window showing none returns a vault with no ID.
type VaultProvider interface {
	GetCurrentVault() domain.Vault
}

// Handler serves TextService over the shown vault. A handler with no text
// returns no live version and refuses an edit.
type Handler struct {
	vaults VaultProvider
	text   FileText
}

// NewHandler is the handler over the text of the shown vault's files.
func NewHandler(vaults VaultProvider, text FileText) *Handler {
	return &Handler{vaults: vaults, text: text}
}

// errNoVault is a call while the window shows no vault.
var errNoVault = errors.New("the window shows no vault")

// ReadLiveText returns the live version of the file and its segments.
func (h *Handler) ReadLiveText(
	ctx context.Context, r *connect.Request[v1.ReadLiveTextRequest],
) (*connect.Response[v1.ReadLiveTextResponse], error) {
	v := h.vaults.GetCurrentVault()
	if h.text == nil || v.ID == "" {
		return connect.NewResponse(&v1.ReadLiveTextResponse{}), nil
	}
	text, ok, err := h.text.ReadLiveText(ctx, v, r.Msg.GetPath())
	if errors.Is(err, fs.ErrNotExist) {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !ok {
		return connect.NewResponse(&v1.ReadLiveTextResponse{}), nil
	}
	return connect.NewResponse(&v1.ReadLiveTextResponse{
		Version: writeVersion(text.Version()), Segments: writeSegments(text.GetSegments()),
	}), nil
}

// CreateTextEdit records the person's edit as a new version in the folder of
// the plugin the request names.
func (h *Handler) CreateTextEdit(
	ctx context.Context, r *connect.Request[v1.CreateTextEditRequest],
) (*connect.Response[v1.CreateTextEditResponse], error) {
	v := h.vaults.GetCurrentVault()
	if h.text == nil || v.ID == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errNoVault)
	}
	version, err := h.text.CreateEdit(ctx, v, textversion.TextEdit{
		PluginID: r.Msg.GetPluginId(), Path: r.Msg.GetPath(), Source: domain.TextVersionID(r.Msg.GetSourceVersionId()),
		Segments: readSegments(r.Msg.GetSegments()),
	})
	switch {
	case err == nil:
		return connect.NewResponse(&v1.CreateTextEditResponse{Version: writeVersion(version)}), nil
	case errors.Is(err, textversion.ErrNoText):
		return connect.NewResponse(&v1.CreateTextEditResponse{Error: v1.TextErrorCode_TEXT_ERROR_CODE_MISSING}), nil
	case errors.Is(err, textversion.ErrInvalidVersion):
		return connect.NewResponse(&v1.CreateTextEditResponse{Error: v1.TextErrorCode_TEXT_ERROR_CODE_INVALID}), nil
	}
	return nil, connect.NewError(connect.CodeInternal, err)
}
