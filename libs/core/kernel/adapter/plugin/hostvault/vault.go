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

// Groups are the capability groups NewAccess serves, each answering its own part of the service.
type Groups struct {
	Files    Files
	Writer   Writer
	Notes    Notes
	Links    Links
	Arranger Arranger
	Stores   Stores
}

// NewAccess returns the capabilities of the open vault the way hostendpoint serves them, each in the
// group that answers it. Each group reads the vault of a call through session.
func NewAccess(session Session, groups Groups) *vault.Access {
	session.rebuild = func(found domain.Vault) *vault.Access {
		return NewAccess(session.showing(found), groups)
	}
	g := groups
	g.Files.session, g.Writer.session, g.Notes.session, g.Links.session, g.Arranger.session, g.Stores.session =
		&session, &session, &session, &session, &session, &session
	return &vault.Access{
		Notes: g.Notes, Links: g.Links, Files: g.Files, Writer: g.Writer, Arranger: g.Arranger, LinkEditor: g.Arranger,
		Watch: &session, Switch: &session, Stores: g.Stores, Cache: g.Stores,
		Locks: g.Writer.Locks,
	}
}
