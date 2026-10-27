package host

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// TextCreation asks for text of a file. Source is the version it is made from, and empty for the
// file itself. IsUserStarted is set for text a person asked for, which runs before text nobody
// asked for.
type TextCreation struct {
	Vault         VaultID
	Path          string
	Source        TextVersionID
	IsUserStarted bool
}

// TextProvider is what a plugin that declares a text input serves to the host. The plugin writes
// each text into its folder with WriteText and records it through Texts.PutVersion; the host reads
// the file itself.
type TextProvider interface {
	// CreateText starts making text, and may return before the text is there.
	CreateText(ctx context.Context, req TextCreation) error
}

// NewTextProviderHandler serves a provider as TextProviderService, for Server.Register.
func NewTextProviderHandler(provider TextProvider, options ...connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewTextProviderServiceHandler(providerService{provider: provider}, options...)
}

type providerService struct {
	provider TextProvider
}

// writeTextError is the code a provider's error is returned with. An error the protocol names no
// code for fails the call.
func writeTextError(err error) (v1.TextErrorCode, error) {
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

func (s providerService) CreateText(
	ctx context.Context, req *connect.Request[v1.TextProviderServiceCreateTextRequest],
) (*connect.Response[v1.TextProviderServiceCreateTextResponse], error) {
	m := req.Msg
	code, err := writeTextError(s.provider.CreateText(ctx, TextCreation{
		Vault: VaultID(m.GetVaultId()), Path: m.GetPath(), Source: TextVersionID(m.GetSourceVersionId()),
		IsUserStarted: m.GetIsUserStarted(),
	}))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.TextProviderServiceCreateTextResponse{Error: code}), nil
}
