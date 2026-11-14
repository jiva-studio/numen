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
	session Session, files Files, writer Writer, notes Notes, links Links, arranger Arranger, stores Stores,
) *vault.Access {
	session.rebuild = func(found domain.Vault) *vault.Access {
		return NewAccess(session.showing(found), files, writer, notes, links, arranger, stores)
	}
	files.session, writer.session, notes.session, links.session, arranger.session, stores.session =
		&session, &session, &session, &session, &session, &session
	return &vault.Access{
		Notes: notes, Links: links, Files: files, Writer: writer, Arranger: arranger, LinkEditor: arranger,
		Watch: &session, Switch: &session, Stores: stores, Cache: stores,
		Locks: writer.Locks,
	}
}
