package catalog

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/restore"
)

// keep writes the verified entries of the catalog to the copy of this device when they are more than the copy holds.
// A server that went back holds fewer, and a log that differs from the copy is left to the recovery: neither
// replaces the copy.
func (r reader) keep(ctx context.Context, entries []*syncv1.CatalogEntry) error {
	server := restore.Journal{Kind: restore.KindCatalog, Entries: make([][]byte, len(entries))}
	for i, entry := range entries {
		raw, err := syncprotocol.EncodeCatalogEntry(entry)
		if err != nil {
			return err
		}
		server.Entries[i] = raw
	}
	cache, err := r.deps.Journals.ReadJournal(ctx, restore.KindCatalog)
	if err != nil {
		return err
	}
	if restore.CompareJournals(cache, server) != restore.ServerAhead {
		return nil
	}
	return r.deps.Journals.WriteJournal(ctx, server)
}
