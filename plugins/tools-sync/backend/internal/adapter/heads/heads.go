// Package heads lists the heads of the live notes of a synced vault the way a pull verifies them.
package heads

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/trust"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

const keySize = 32

// errUnknownRemoteVault means the account has no synced vault of the id.
var errUnknownRemoteVault = errors.New("the account has no such synced vault")

// Deps are what the adapter needs.
type Deps struct {
	Catalog   ports.Catalog
	Chain     ports.Chain
	Revisions ports.Revisions
	Crypto    ports.Crypto
	Identity  ports.Identity
}

// Heads reads the heads of a synced vault from the server.
type Heads struct{ deps Deps }

// New returns the adapter over the dependencies.
func New(deps Deps) *Heads { return &Heads{deps: deps} }

// ListHeads reads every page from the first revision, checks each revision like a pull and returns the live ones. A
// revision that does not verify is left out.
func (h *Heads) ListHeads(ctx context.Context, remoteID string) ([]ports.Head, error) {
	vaultID, err := hex.DecodeString(remoteID)
	if err != nil {
		return nil, errUnknownRemoteVault
	}
	authors, err := h.getAuthors(ctx)
	if err != nil {
		return nil, err
	}
	keys, err := h.getKeys(ctx, vaultID, authors)
	if err != nil {
		return nil, err
	}
	var listed []ports.Head
	for since := uint64(0); ; {
		page, err := h.deps.Revisions.GetChanges(ctx, vaultID, since, 0)
		if err != nil {
			return nil, err
		}
		listed = appendHeads(listed, page.GetChanges(), keys, authors)
		if !page.GetHasMore() {
			return listed, nil
		}
		since = page.GetNextSeq()
	}
}

func appendHeads(listed []ports.Head, changes []*syncv1.StoredRevision, keys map[uint32][keySize]byte, authors trust.Authors) []ports.Head {
	for _, stored := range changes {
		if stored.GetRevision().GetDeleted() {
			continue
		}
		key, isKnown := keys[stored.GetRevision().GetEpoch()]
		if !isKnown {
			continue
		}
		meta, err := note.OpenRevision(key, stored, authors.IsTrusted)
		if err != nil {
			continue
		}
		listed = append(listed, ports.Head{Path: meta.Path, Hash: meta.Hash})
	}
	return listed
}

func (h *Heads) getAuthors(ctx context.Context) (trust.Authors, error) {
	own, err := h.deps.Identity.GetPublicIdentity(ctx)
	if err != nil {
		return trust.Authors{}, err
	}
	stored, err := h.deps.Chain.GetAccountChain(ctx)
	if err != nil {
		return trust.Authors{}, err
	}
	return trust.NewAuthors(own.Chain, stored)
}

func (h *Heads) getKeys(ctx context.Context, vaultID []byte, authors trust.Authors) (map[uint32][keySize]byte, error) {
	entries, err := h.deps.Catalog.GetCatalog(ctx, 0)
	if err != nil {
		return nil, err
	}
	if err := vaultlink.CheckEntries(entries, authors); err != nil {
		return nil, err
	}
	var last *syncv1.CatalogEntry
	for _, entry := range entries {
		if string(entry.GetVaultId()) == string(vaultID) {
			last = entry
		}
	}
	if last == nil {
		return nil, errUnknownRemoteVault
	}
	plain, err := h.deps.Crypto.OpenData(ctx, vaultlink.GetCatalogContext(vaultID, last.GetVersion()), last.GetEncBody())
	if err != nil {
		return nil, fmt.Errorf("open the body of catalog entry %d: %w", last.GetVersion(), err)
	}
	body, err := vaultlink.ParseCatalogBody(plain)
	if err != nil {
		return nil, err
	}
	keys := map[uint32][keySize]byte{}
	for _, one := range body.GetKeys() {
		if len(one.Key) != keySize {
			return nil, errUnknownRemoteVault
		}
		keys[one.Epoch] = [keySize]byte(one.Key)
	}
	return keys, nil
}
