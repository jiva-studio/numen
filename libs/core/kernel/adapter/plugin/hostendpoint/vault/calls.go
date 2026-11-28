package vault

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

// Header is the header of a call that names the vault it works, by identity.
const Header = "Numen-Vault"

// Authority tells which plugin made a call.
type Authority interface {
	// Authenticate returns the id of the plugin the Authorization header names.
	Authenticate(authorization string) (pluginID string, err error)
	// IsTakingFile reports whether the plugin takes the file at the path: a plugin whose manifest declares
	// onFilesChanged with a mask takes the types it matches, and one that declares none takes every file.
	IsTakingFile(pluginID, path string, noteTypes []domain.NoteType) bool
}

// Source lends the vault the host works and the list of vaults a plugin changes.
type Source interface {
	GetPluginVault() *Access
	GetPluginVaultList() List
}

// Select returns the vault a call works: the one the header names, or the one the host shows.
func Select(access *Access, header http.Header) (*Access, error) {
	named := header.Get(Header)
	if access == nil || named == "" {
		return access, nil
	}
	linked, err := access.Switch.With(domain.VaultID(named))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return linked, nil
}

// calls answers what every call of the service asks first: who calls and which vault it works.
type calls struct {
	auth   Authority
	source Source
}

func (c *calls) begin(header http.Header) (*Access, error) {
	_, access, err := c.beginAs(header)
	return access, err
}

func (c *calls) beginAs(header http.Header) (string, *Access, error) {
	pluginID, err := c.auth.Authenticate(header.Get("Authorization"))
	if err != nil {
		return "", nil, err
	}
	return c.selectVault(pluginID, header)
}

func (c *calls) beginOwnData(header http.Header) (string, *Access, error) {
	pluginID, err := c.auth.Authenticate(header.Get("Authorization"))
	if err != nil {
		return "", nil, err
	}
	return c.selectVault(pluginID, header)
}

func (c *calls) selectVault(pluginID string, header http.Header) (string, *Access, error) {
	access, err := Select(c.source.GetPluginVault(), header)
	if err != nil {
		return "", nil, err
	}
	return pluginID, access, nil
}

func serve[Q, R any](
	ctx context.Context, c *calls, req *connect.Request[Q],
	withCode func(code v1.ErrorCode) *R, work func(access *Access) (*R, error),
) (*connect.Response[R], error) {
	access, err := c.begin(req.Header())
	if err != nil {
		return nil, err
	}
	if access == nil {
		return connect.NewResponse(withCode(noVault)), nil
	}
	out, err := work(access)
	if err == nil {
		return connect.NewResponse(out), nil
	}
	code, failure := answer(ctx, err)
	if failure != nil {
		return nil, failure
	}
	return connect.NewResponse(withCode(code)), nil
}
