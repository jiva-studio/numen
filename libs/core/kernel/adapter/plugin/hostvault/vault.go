package hostvault

import (
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
)

// Vault is the open vault as the host exposes it to a plugin, over the use cases and the queries of
// the installation. Access returns the capabilities hostendpoint serves, each in the group that
// answers it.
type Vault struct {
	Session     Session
	Files       Files
	Lookups     Lookups
	Connections Connections
	Arranger    Arranger
	Stores      Stores
	Locks       vault.Locks
}

// Access returns the capabilities of the vault the way hostendpoint serves them.
func (v Vault) Access() *vault.Access {
	session := v.Session
	session.rebuild = func(found domain.Vault) *vault.Access {
		linked := v
		linked.Session = v.Session.showing(found)
		return linked.Access()
	}
	files, lookups, connections, arranger, stores := v.Files, v.Lookups, v.Connections, v.Arranger, v.Stores
	files.session, lookups.session, connections.session, arranger.session, stores.session =
		&session, &session, &session, &session, &session
	return &vault.Access{
		Notes: lookups, Links: lookups, Files: files, Writer: files, Arranger: arranger, LinkEditor: arranger,
		Connections: connections, Related: connections, Watch: &session, Switch: &session, Stores: stores, Cache: stores,
		Locks: v.Locks,
	}
}
