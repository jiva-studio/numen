// Package restore brings the notes of this device back to a sync server that was restored from a backup: it
// checks the epoch the server announced, replays the logs of the account from the copy of this device and
// sends the notes the server lost.
package restore

import (
	"bytes"
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	domain "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/restore"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// ErrKeyChanged is domain.ErrKeyChanged: the key that signs restore epochs is not the key this device pinned.
var ErrKeyChanged = domain.ErrKeyChanged

// ErrDiverged is domain.ErrDiverged: a log on the server differs from the copy of this device, so nothing is replayed.
var ErrDiverged = domain.ErrDiverged

// Committer commits notes to the server.
type Committer interface {
	Execute(ctx context.Context, in ports.CommitInput) ([]transfer.Outcome, error)
}

// Remover commits the removal of notes to the server.
type Remover interface {
	Execute(ctx context.Context, in transfer.RemoveInput) ([]transfer.Outcome, error)
}

// Deps are what the use cases need. Address is the address of the server, the name its key is pinned under.
type Deps struct {
	Address   string
	Epochs    ports.Epochs
	Mirror    ports.Mirror
	Catalog   ports.Catalog
	Chain     ports.Chain
	Revisions ports.Revisions
	Blobs     ports.BlobDownloader
	Crypto    ports.Crypto
	Host      ports.Host
	State     State
	Commit    Committer
	Remove    Remover
}

// State is the link and the records of a vault.
type State interface {
	ports.Links
	ports.Records
}

// RecoverAccount checks the epoch of the server against the key this device pinned, pins the key the first time
// it sees one, signs in again with the chain of this device and sends the catalog entries the server lost.
type RecoverAccount struct{ deps Deps }

// NewRecoverAccount returns the use case over the dependencies.
func NewRecoverAccount(deps Deps) *RecoverAccount { return &RecoverAccount{deps: deps} }

// Execute returns ErrKeyChanged for a key that is not the pinned one and ErrDiverged for a log that differs from
// the copy, and sends nothing in either case.
func (u *RecoverAccount) Execute(ctx context.Context) error {
	_, key, err := u.deps.Epochs.GetRestoreEpoch(ctx)
	if err != nil {
		return err
	}
	if err := u.pin(ctx, key); err != nil {
		return err
	}
	if err := u.deps.Chain.SendAccountChain(ctx); err != nil {
		return err
	}
	return u.replayCatalog(ctx)
}

func (u *RecoverAccount) pin(ctx context.Context, key []byte) error {
	pinned, err := u.deps.Mirror.ReadPin(ctx, u.deps.Address)
	if err != nil {
		return err
	}
	switch {
	case len(pinned) == 0:
		return u.deps.Mirror.WritePin(ctx, u.deps.Address, key)
	case !bytes.Equal(pinned, key):
		return ErrKeyChanged
	}
	return nil
}

func (u *RecoverAccount) replayCatalog(ctx context.Context) error {
	cache, err := u.deps.Mirror.ReadJournal(ctx, domain.KindCatalog)
	if err != nil || len(cache.Entries) == 0 {
		return err
	}
	entries, err := u.deps.Catalog.GetCatalog(ctx, 0)
	if err != nil {
		return err
	}
	server := domain.Journal{Kind: domain.KindCatalog, Entries: make([][]byte, len(entries))}
	for i, entry := range entries {
		if server.Entries[i], err = syncprotocol.EncodeCatalogEntry(entry); err != nil {
			return err
		}
	}
	replay, err := domain.GetReplay(cache, server)
	if err != nil {
		return err
	}
	for _, raw := range replay {
		entry, err := syncprotocol.DecodeCatalogEntry(raw)
		if err != nil {
			return err
		}
		response, err := u.deps.Catalog.AppendCatalogEntry(ctx, entry)
		if err != nil {
			return err
		}
		if !response.GetCommitted() {
			return fmt.Errorf("%w: another device appended entry %d first", ErrDiverged, entry.GetVersion())
		}
	}
	return nil
}
