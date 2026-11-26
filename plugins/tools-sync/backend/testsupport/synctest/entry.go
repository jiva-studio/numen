package synctest

import (
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

func (c *Computer) NewCatalogEntry(t *testing.T, prev *syncv1.CatalogEntry, vaultID []byte, isCreating bool, body []byte) *syncv1.CatalogEntry {
	t.Helper()
	entry := &syncv1.CatalogEntry{Version: 1, VaultId: vaultID, CreatesVault: isCreating, EncBody: body}
	if prev != nil {
		id := syncprotocol.GetCatalogEntryID(prev)
		entry.Version, entry.PrevHash = prev.GetVersion()+1, id[:]
	}
	c.SignCatalogEntry(t, entry)
	return entry
}

func (c *Computer) SignCatalogEntry(t *testing.T, entry *syncv1.CatalogEntry) {
	t.Helper()
	public, err := c.Account.GetPublicIdentity(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	entry.AuthorDeviceSignKey = public.DeviceSignKey
	signature, err := c.Account.SignAsDevice(t.Context(), syncprotocol.GetCatalogEntryMessage(entry))
	if err != nil {
		t.Fatal(err)
	}
	entry.Signature = signature
}
