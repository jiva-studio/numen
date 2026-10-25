package host

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/notes/domain"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// TextRequest names one page of a text the plugin keeps. Cursor is the NextCursor of the page
// before, and empty for the first.
type TextRequest struct {
	Vault  domain.VaultID
	Path   string
	Key    string
	Cursor string
}

// TextCreation asks for text of a file. Input is the version it is made from, and empty for the
// file itself. IsUserStarted is set for text a person asked for, which runs before text nobody
// asked for.
type TextCreation struct {
	Vault         domain.VaultID
	Path          string
	Input         domain.TextVersionID
	IsUserStarted bool
}

// TextProvider is what a plugin that declares a text input answers the host with. The plugin keeps
// each of its texts under a key of its own, and reports each through Texts.PutVersion.
type TextProvider interface {
	// GetText is one page of a text the plugin keeps, or ErrNotExist.
	GetText(ctx context.Context, req TextRequest) (TextPage, error)
	// CreateText starts making text, and may return before the text is there.
	CreateText(ctx context.Context, req TextCreation) error
	// DeleteText removes a text the plugin keeps. A text that is not there is removed.
	DeleteText(ctx context.Context, vault domain.VaultID, path, key string) error
}

// NewTextProviderHandler serves a provider as TextProviderService, for Server.Register.
func NewTextProviderHandler(provider TextProvider, options ...connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewTextProviderServiceHandler(providerService{provider: provider}, options...)
}

type providerService struct {
	provider TextProvider
}

// answerText is the code a provider's error is answered with. An error the protocol names no code
// for fails the call.
func answerText(err error) (v1.TextErrorCode, error) {
	switch {
	case err == nil:
		return v1.TextErrorCode_TEXT_ERROR_CODE_UNSPECIFIED, nil
	case errors.Is(err, ErrNotExist):
		return v1.TextErrorCode_TEXT_ERROR_CODE_MISSING, nil
	case errors.Is(err, ErrNoVault):
		return v1.TextErrorCode_TEXT_ERROR_CODE_NO_VAULT, nil
	case errors.Is(err, ErrInvalidVersion):
		return v1.TextErrorCode_TEXT_ERROR_CODE_INVALID, nil
	}
	return 0, err
}

func (s providerService) GetText(
	ctx context.Context, req *connect.Request[v1.TextProviderServiceGetTextRequest],
) (*connect.Response[v1.TextProviderServiceGetTextResponse], error) {
	m := req.Msg
	page, err := s.provider.GetText(ctx, TextRequest{
		Vault: domain.VaultID(m.GetVaultId()), Path: m.GetPath(), Key: m.GetKey(), Cursor: m.GetCursor(),
	})
	code, err := answerText(err)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.TextProviderServiceGetTextResponse{
		Error: code, Parts: writeTextParts(page.Parts), NextCursor: page.NextCursor,
	}), nil
}

func (s providerService) CreateText(
	ctx context.Context, req *connect.Request[v1.TextProviderServiceCreateTextRequest],
) (*connect.Response[v1.TextProviderServiceCreateTextResponse], error) {
	m := req.Msg
	code, err := answerText(s.provider.CreateText(ctx, TextCreation{
		Vault: domain.VaultID(m.GetVaultId()), Path: m.GetPath(), Input: domain.TextVersionID(m.GetInputVersionId()),
		IsUserStarted: m.GetIsUserStarted(),
	}))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.TextProviderServiceCreateTextResponse{Error: code}), nil
}

func (s providerService) DeleteText(
	ctx context.Context, req *connect.Request[v1.TextProviderServiceDeleteTextRequest],
) (*connect.Response[v1.TextProviderServiceDeleteTextResponse], error) {
	m := req.Msg
	code, err := answerText(s.provider.DeleteText(ctx, domain.VaultID(m.GetVaultId()), m.GetPath(), m.GetKey()))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.TextProviderServiceDeleteTextResponse{Error: code}), nil
}
