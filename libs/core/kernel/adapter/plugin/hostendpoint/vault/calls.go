package vault

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// Header is the header of a call that names the vault it works, by identity.
const Header = "Numen-Vault"

// Authority tells which plugin made a call and whether it holds a permission.
type Authority interface {
	// Authorize returns the id of the plugin the Authorization header names, and refuses a plugin
	// without the permission.
	Authorize(authorization, permission string) (pluginID string, err error)
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

func (c *calls) begin(header http.Header, permission string) (string, *Access, error) {
	pluginID, err := c.auth.Authorize(header.Get("Authorization"), permission)
	if err != nil {
		return "", nil, err
	}
	access, err := Select(c.source.GetPluginVault(), header)
	if err != nil {
		return "", nil, err
	}
	return pluginID, access, nil
}

func serve[Q, R any](
	ctx context.Context, c *calls, req *connect.Request[Q], permission string,
	withCode func(code v1.HostVaultErrorCode) *R, work func(access *Access) (*R, error),
) (*connect.Response[R], error) {
	_, access, err := c.begin(req.Header(), permission)
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
