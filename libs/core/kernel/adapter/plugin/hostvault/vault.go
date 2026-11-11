package hostvault

import (
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
)

// sessionAccess is what the groups of the vault read of the session: the vault of a call, and the
// notice that a call changed a file.
type sessionAccess interface {
	open() (domain.Vault, error)
	notifyFileWritten()
	notifyAfterWrite(err error) error
}

// NewAccess returns the capabilities of the open vault the way hostendpoint serves them, each in the
// group that answers it. Each group reads the vault of a call through session.
func NewAccess(
	session Session, files Files, lookups Lookups, connections Connections, arranger Arranger, stores Stores, locks vault.Locks,
) *vault.Access {
	session.rebuild = func(found domain.Vault) *vault.Access {
		return NewAccess(session.showing(found), files, lookups, connections, arranger, stores, locks)
	}
	files.session, lookups.session, connections.session, arranger.session, stores.session =
		&session, &session, &session, &session, &session
	return &vault.Access{
		Notes: lookups, Links: lookups, Files: files, Writer: files, Arranger: arranger, LinkEditor: arranger,
		Connections: connections, Related: connections, Watch: &session, Switch: &session, Stores: stores, Cache: stores,
		Locks: locks,
	}
}
