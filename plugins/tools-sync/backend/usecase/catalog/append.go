package catalog

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
)

type entryInput struct {
	vaultID    []byte
	isCreating bool
	body       vaultlink.CatalogBody
}

func (r reader) appendEntry(ctx context.Context, in entryInput) error {
	for range maxTries {
		entries, err := r.read(ctx)
		if err != nil {
			return err
		}
		entry, err := r.newEntry(ctx, entries, in)
		if err != nil {
			return err
		}
		response, err := r.deps.Catalog.AppendCatalogEntry(ctx, entry)
		if err != nil {
			return err
		}
		if response.GetCommitted() {
			return r.keep(ctx, append(entries, entry))
		}
	}
	return ErrCatalogBusy
}

func (r reader) newEntry(ctx context.Context, entries []*syncv1.CatalogEntry, in entryInput) (*syncv1.CatalogEntry, error) {
	public, err := r.deps.Identity.GetPublicIdentity(ctx)
	if err != nil {
		return nil, err
	}
	entry := &syncv1.CatalogEntry{
		Version: uint64(len(entries)) + 1, VaultId: in.vaultID, CreatesVault: in.isCreating, AuthorDeviceSignKey: public.DeviceSignKey,
	}
	if last := len(entries) - 1; last >= 0 {
		previous := syncprotocol.GetCatalogEntryID(entries[last])
		entry.PrevHash = previous[:]
	}
	plain, err := in.body.Encode()
	if err != nil {
		return nil, err
	}
	if entry.EncBody, err = r.deps.Crypto.SealData(ctx, vaultlink.GetCatalogContext(in.vaultID, entry.GetVersion()), plain); err != nil {
		return nil, fmt.Errorf("seal the body of a catalog entry: %w", err)
	}
	if entry.Signature, err = r.deps.Identity.SignAsDevice(ctx, syncprotocol.GetCatalogEntryMessage(entry)); err != nil {
		return nil, err
	}
	return entry, nil
}
