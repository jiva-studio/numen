// Package restore brings the notes of this device back to a sync server that was restored from a backup: it
// checks the epoch the server announced, replays the logs of the account from the copy of this device and
// sends the notes the server lost.
package restore

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	domain "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/restore"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// ErrEpochUntrusted is domain.ErrEpochUntrusted: the announcement of the server has no valid certificate of a known root key.
var ErrEpochUntrusted = domain.ErrEpochUntrusted

// ErrDiverged is domain.ErrDiverged: a log on the server differs from the copy of this device, so nothing is replayed.
var ErrDiverged = domain.ErrDiverged

// ErrForeignJournal is domain.ErrForeignJournal: the copy of the catalog on this device holds an entry of another identity; it was dropped and nothing is replayed.
var ErrForeignJournal = domain.ErrForeignJournal

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
	Identity  ports.Identity
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

// RecoverAccount requires an announcement that a known root key vouches for, unless the server holds no catalog entry at all, signs in again with the chain of this device and sends the catalog entries the server lost.
type RecoverAccount struct{ deps Deps }

// NewRecoverAccount returns the use case over the dependencies.
func NewRecoverAccount(deps Deps) *RecoverAccount { return &RecoverAccount{deps: deps} }

// Execute returns ErrEpochUntrusted for a missing announcement of a server that holds catalog entries, a forged or an expired one and ErrDiverged for a log that differs from
// the copy, and sends nothing in either case.
func (u *RecoverAccount) Execute(ctx context.Context) error {
	epoch, _, err := u.deps.Epochs.GetRestoreEpoch(ctx)
	if err != nil {
		return err
	}
	if epoch == nil {
		if err := u.requireEmptyCatalog(ctx); err != nil {
			return err
		}
	}
	if err := u.deps.Chain.SendAccountChain(ctx); err != nil {
		return err
	}
	return u.replayCatalog(ctx)
}

func (u *RecoverAccount) requireEmptyCatalog(ctx context.Context) error {
	entries, err := u.deps.Catalog.GetCatalog(ctx, 0)
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		return fmt.Errorf("%w: the server announced no restore", ErrEpochUntrusted)
	}
	return nil
}

func (u *RecoverAccount) replayCatalog(ctx context.Context) error {
	cache, err := u.deps.Mirror.ReadJournal(ctx, domain.KindCatalog)
	if err != nil || len(cache.Entries) == 0 {
		return err
	}
	if err := u.checkAuthors(ctx, cache); err != nil {
		return u.dropForeign(ctx, err)
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

func (u *RecoverAccount) checkAuthors(ctx context.Context, cache domain.Journal) error {
	raw, err := u.deps.Chain.GetAccountChain(ctx)
	if err != nil {
		return err
	}
	chain, err := identity.ParseChain(raw)
	if err != nil {
		return fmt.Errorf("read the chain of the account: %w", err)
	}
	for _, encoded := range cache.Entries {
		entry, err := syncprotocol.DecodeCatalogEntry(encoded)
		if err != nil {
			return err
		}
		if !chain.IsActiveDevice(entry.GetAuthorDeviceSignKey()) {
			return domain.ErrForeignJournal
		}
	}
	return nil
}

func (u *RecoverAccount) dropForeign(ctx context.Context, cause error) error {
	if !errors.Is(cause, domain.ErrForeignJournal) {
		return cause
	}
	if err := u.deps.Mirror.WriteJournal(ctx, domain.Journal{Kind: domain.KindCatalog, Entries: nil}); err != nil {
		return err
	}
	return cause
}
