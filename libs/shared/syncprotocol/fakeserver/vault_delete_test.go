package fakeserver_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

func (e *env) getUsage() map[string]uint64 {
	e.tb.Helper()
	usage, err := e.catalog.GetVaultUsage(e.tb.Context())
	if err != nil {
		e.tb.Fatal(err)
	}
	sizes := map[string]uint64{}
	for _, one := range usage {
		sizes[string(one.GetVaultId())] = one.GetUsedBytes()
	}
	return sizes
}

func TestGetVaultUsage_ListsEveryVaultWithItsReservedAndCommittedBytes(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	filled, empty := e.newVault(), e.newVault()
	e.commit(filled, e.newFile(filled, newID(16), nil, "a.md", []byte("12345")))
	e.reserveOnly(filled, 7)

	got := e.getUsage()

	if len(got) != 2 || got[string(filled)] != 12 || got[string(empty)] != 0 {
		t.Errorf("usage = %v, want the filled vault at 12 and the empty one at 0", got)
	}
}

func TestGetVaultUsage_DoesNotCountAReplacedBlobAndAddsUpToTheUsedBytes(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, false)
	e := b.env
	vault, file := e.newVault(), newID(16)
	first := e.newFile(vault, file, nil, "a.md", []byte("one"))
	e.commit(vault, first)
	parent := syncprotocol.GetRevisionID(first)
	second := e.newFile(vault, file, parent[:], "a.md", []byte("second body"))
	e.commit(vault, second)

	got := e.getUsage()

	if got[string(vault)] != 11 || b.subscriptionNow().UsedBytes != 11 {
		t.Errorf("usage = %v, used = %d, want 11 in both", got, b.subscriptionNow().UsedBytes)
	}
}

func TestGetVaultUsage_ListsOnlyTheVaultsOfTheAccount(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.newVault()
	other := newSecondAccountOn(t, e.address)

	if got := other.getUsage(); len(got) != 0 {
		t.Errorf("usage = %v, want none", got)
	}
}

func TestDeleteVault_FreesBytes(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, false)
	e := b.env
	vault, kept := e.newVault(), e.newVault()
	e.commit(vault, e.newFile(vault, newID(16), nil, "a.md", []byte("12345")))
	e.commit(kept, e.newFile(kept, newID(16), nil, "b.md", []byte("123")))

	got, err := e.catalog.DeleteVault(t.Context(), vault)

	if err != nil || !got.GetDeleted() || got.GetFreedBytes() != 5 {
		t.Fatalf("DeleteVault() = %v, %v, want deleted and 5 bytes freed", got, err)
	}
	if usage := e.getUsage(); len(usage) != 1 || usage[string(kept)] != 3 || b.subscriptionNow().UsedBytes != 3 {
		t.Errorf("usage = %v, used = %d, want only the other vault at 3", usage, b.subscriptionNow().UsedBytes)
	}
	if _, err := e.revision.GetChanges(t.Context(), vault, 0, 10); !errors.Is(err, syncprotocol.ErrNoVault) {
		t.Errorf("GetChanges after the delete = %v, want ErrNoVault", err)
	}
}

func TestDeleteVault_RefusesAReserveAndACommitAfterTheDelete(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	upload := e.upload(vault, []byte("body"))
	revision := e.newRevisionOf(vault, newID(16), nil, "a.md", upload, []byte("body"))
	if _, err := e.catalog.DeleteVault(t.Context(), vault); err != nil {
		t.Fatal(err)
	}

	_, reserveErr := e.blob.ReserveBlobs(t.Context(), vault, []uint64{8})
	_, commitErr := e.revision.CommitRevisions(t.Context(), vault, []*syncv1.Revision{revision})

	if !errors.Is(reserveErr, syncprotocol.ErrNoVault) || !errors.Is(commitErr, syncprotocol.ErrNoVault) {
		t.Errorf("reserve = %v, commit = %v, want ErrNoVault for both", reserveErr, commitErr)
	}
}

func TestDeleteVault_ReportsNotDeletedForUnknownVault(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	other := newSecondAccountOn(t, e.address)
	e.commit(vault, e.newFile(vault, newID(16), nil, "a.md", []byte("12345")))

	foreign, foreignErr := other.catalog.DeleteVault(t.Context(), vault)
	first, firstErr := e.catalog.DeleteVault(t.Context(), vault)
	again, againErr := e.catalog.DeleteVault(t.Context(), vault)
	unknown, unknownErr := e.catalog.DeleteVault(t.Context(), newID(16))

	if foreignErr != nil || foreign.GetDeleted() || foreign.GetFreedBytes() != 0 {
		t.Errorf("a foreign vault = %v, %v, want OK and not deleted", foreign, foreignErr)
	}
	if firstErr != nil || !first.GetDeleted() {
		t.Errorf("first delete = %v, %v, want deleted", first, firstErr)
	}
	for name, got := range map[string]struct {
		response *syncv1.DeleteVaultResponse
		err      error
	}{"repeat": {again, againErr}, "unknown": {unknown, unknownErr}} {
		if got.err != nil || got.response.GetDeleted() || got.response.GetFreedBytes() != 0 {
			t.Errorf("%s = %v, %v, want OK and not deleted", name, got.response, got.err)
		}
	}
}

func TestDeleteVault_IsAllowedInReadOnlyAndRefusedInDeleted(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, false)
	vault, second := b.env.newVault(), b.env.newVault()

	b.setPhase("phase=read_only")
	if got, err := b.env.catalog.DeleteVault(t.Context(), vault); err != nil || !got.GetDeleted() {
		t.Errorf("read only: DeleteVault() = %v, %v, want deleted", got, err)
	}
	b.setPhase("phase=deleted")
	if _, err := b.env.catalog.DeleteVault(t.Context(), second); !errors.Is(err, syncprotocol.ErrWriteNotAllowed) {
		t.Errorf("deleted: DeleteVault() = %v, want ErrWriteNotAllowed", err)
	}
}

func TestDeleteVault_RemovesTheBodiesOfTheVault(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	upload := e.upload(vault, []byte("body"))

	if _, err := e.catalog.DeleteVault(t.Context(), vault); err != nil {
		t.Fatal(err)
	}

	if stored := e.getAdmin("/admin/blobs"); bytes.Contains(stored, upload.GetBlobId()) {
		t.Errorf("the blob %x is still stored after the vault was deleted", upload.GetBlobId())
	}
}
