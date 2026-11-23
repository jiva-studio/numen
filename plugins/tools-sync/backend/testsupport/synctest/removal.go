package synctest

import (
	"crypto/rand"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
)

// CommitRemoval commits a removal of the note as another computer does: a signed revision with no blob, over
// the revision the outcome names.
func (c *Computer) CommitRemoval(t *testing.T, vault transfer.Vault, notePath string, previous transfer.Outcome) {
	t.Helper()
	public, err := c.Account.GetPublicIdentity(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	key, err := note.NewID(rand.Reader, 32)
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := note.SealMeta(rand.Reader, vault.Key, vault.ID, previous.FileID, note.Meta{Path: notePath, Key: key, Kind: note.KindMarkdown})
	if err != nil {
		t.Fatal(err)
	}
	revision := &syncv1.Revision{
		VaultId: vault.ID, FileId: previous.FileID, ParentRev: previous.RevID, Deleted: true, EncMeta: sealed,
		PathHash: note.PathHash(vault.Key, notePath), AuthorDeviceSignKey: public.DeviceSignKey,
	}
	if revision.Signature, err = c.Account.SignAsDevice(t.Context(), syncprotocol.GetRevisionMessage(revision)); err != nil {
		t.Fatal(err)
	}
	response, err := c.Cloud.CommitRevisions(t.Context(), vault.ID, []*syncv1.Revision{revision})
	if err != nil {
		t.Fatal(err)
	}
	if !response.GetCommitted() {
		t.Fatalf("the removal of %s was not committed: %v", notePath, response.GetConflicts())
	}
}

// GetDeviceSignKey returns the public signing key of this computer.
func (c *Computer) GetDeviceSignKey(t *testing.T) []byte {
	t.Helper()
	public, err := c.Account.GetPublicIdentity(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return public.DeviceSignKey
}
