package vault

import (
	"net/http"

	"connectrpc.com/connect"

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
