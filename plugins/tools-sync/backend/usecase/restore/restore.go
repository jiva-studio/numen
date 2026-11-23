// Package restore brings the notes of this device back to a sync server that was restored from a backup: it
// checks the epoch the server announced, replays the logs of the account from the copy of this device and
// sends the notes the server lost.
package restore

import (
	"context"
	"errors"
	"io"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

var errNotImplemented = errors.New("not implemented")

// ErrKeyChanged means the key that signs restore epochs is not the key this device pinned.
var ErrKeyChanged = errors.New("the restore key of the sync server changed")

// ErrDiverged means a log on the server differs from the copy of this device, so nothing is replayed.
var ErrDiverged = errors.New("the sync server holds another history than this device")

// Deps are what the use cases need. Address is the address of the server, the name its key is pinned under.
type Deps struct {
	Address   string
	Epochs    ports.Epochs
	Pins      ports.Pins
	Journals  ports.Journals
	Catalog   ports.Catalog
	Chain     ports.Chain
	Revisions ports.Revisions
	Uploader  ports.BlobUploader
	Crypto    ports.Crypto
	Identity  ports.Identity
	Host      ports.Host
	State     State
	Health    ports.Health
	Random    io.Reader
}

// State is the link and the records of a vault.
type State interface {
	ports.Links
	ports.Records
}

// RecoverAccount checks the epoch of the server against the key this device pinned, pins the key the first time
// it sees one, and sends the logs of the account the server lost: the catalog entries after the head of the
// server and the chain of devices.
type RecoverAccount struct{ deps Deps }

// NewRecoverAccount returns the use case over the dependencies.
func NewRecoverAccount(deps Deps) *RecoverAccount { return &RecoverAccount{deps: deps} }

// Execute returns ErrKeyChanged for a key that is not the pinned one, ErrDiverged for a log that differs from the
// copy, and sends nothing in either case.
func (u *RecoverAccount) Execute(ctx context.Context) error { return errNotImplemented }

// RecoverVault sends the notes of one vault the server lost or holds in an older state, and writes the epoch id to
// the link only after the last note was examined.
type RecoverVault struct{ deps Deps }

// NewRecoverVault returns the use case over the dependencies.
func NewRecoverVault(deps Deps) *RecoverVault { return &RecoverVault{deps: deps} }

// Execute runs the recovery of the vault, continuing what an earlier run left in the link.
func (u *RecoverVault) Execute(ctx context.Context, id domain.VaultID) error {
	return errNotImplemented
}
