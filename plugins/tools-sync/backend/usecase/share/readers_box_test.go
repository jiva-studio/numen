package share_test

import (
	"context"
	"crypto/rand"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	domainshare "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/chaincourier"
	usecase "github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/share"
)

func TestAcceptInvitation_AReaderRestoredFromTheWordsOpensTheSealOfItsEarlierBox(t *testing.T) {
	w := newWorld(t)
	words := synctest.NewWords(t)
	reader := synctest.NewFirstComputer(t, w.server, w.clock, "Reader", words)
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots")
	w.syncOwner(t)
	w.shareFolder(t)
	w.syncOwner(t)
	invitation := w.addReader(t, reader, "Anna")
	restored := synctest.NewRestoredComputer(t, w.server, w.clock, "Restored", words)

	received, err := restored.AcceptInvitation(t.Context(), invitation.Text)

	if err != nil {
		t.Fatalf("the restored device could not take the share: %v", err)
	}
	if got, ok := restored.ReadNote(received.VaultID, "Soup.md"); !ok || got != "carrots" {
		t.Errorf("the note on the restored device = %q (%v), want carrots", got, ok)
	}
}

// revokeADevice makes the reader add a device and revoke it, which moves its account to a new box key, and returns the keys that device held.
func revokeADevice(t *testing.T, w *world) identity.AccountKeys {
	t.Helper()
	stranger, err := identity.NewDeviceKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.reader.Fake.AddDeviceAt(stranger.Public(), "Phone", w.clock.Now()); err != nil {
		t.Fatal(err)
	}
	held, err := w.reader.Fake.GetAccountKeys()
	if err != nil {
		t.Fatal(err)
	}
	if err := w.reader.Fake.RevokeDevice(stranger.Public().Sign); err != nil {
		t.Fatal(err)
	}
	if published, err := chaincourier.NewPublish(w.reader.ChainDeps()).Execute(t.Context()); err != nil || !published {
		t.Fatalf("published = %v, error = %v, want the revoke published", published, err)
	}
	return held
}

func newestSeal(t *testing.T, w *world, reader []byte) (shareID []byte, epoch uint32, sealed []byte) {
	t.Helper()
	shares, err := w.owner.Cloud.ListShares(t.Context())
	if err != nil || len(shares) != 1 {
		t.Fatalf("shares = %v, error = %v, want one", shares, err)
	}
	entries, _, err := w.owner.Cloud.GetShareLog(t.Context(), shares[0].GetShareRootId(), 0)
	if err != nil {
		t.Fatal(err)
	}
	last := entries[len(entries)-1]
	for _, seal := range last.GetSeals() {
		if string(seal.GetReaderRootSignKey()) == string(reader) {
			return shares[0].GetShareRootId(), last.GetEpoch(), seal.GetSealed()
		}
	}
	t.Fatalf("the newest entry of the share log has no seal for the reader: %v", last.GetKind())
	return nil, 0, nil
}

func openWith(keys identity.AccountKeys) domainshare.Opener {
	return func(_ context.Context, boxContext, sealed []byte) ([]byte, error) {
		return keys.OpenSealedBox(syncprotocol.SignInPluginID, boxContext, sealed)
	}
}

func readerRootOf(t *testing.T, c *synctest.Computer) []byte {
	t.Helper()
	chain, err := identity.ParseChain(c.Fake.Public().Chain)
	if err != nil {
		t.Fatal(err)
	}
	return chain.Root().Sign
}

func TestRefreshReader_SealsTheNewEpochToTheBoxOfAReaderThatRevokedADevice(t *testing.T) {
	w := newWorld(t)
	w.shareFolder(t)
	w.addReader(t, w.reader, "Anna")
	held := revokeADevice(t, w)
	code, err := w.reader.GetReaderCode(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if err := usecase.NewRefreshReader(w.owner.ShareDeps()).Execute(t.Context(), owned, folder, code.Code); err != nil {
		t.Fatalf("RefreshReader() = %v", err)
	}

	root := readerRootOf(t, w.reader)
	shareID, epoch, sealed := newestSeal(t, w, root)
	current, err := w.reader.Fake.GetAccountKeys()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := domainshare.OpenSeal(t.Context(), openWith(current), shareID, root, epoch, sealed); err != nil {
		t.Errorf("the reader cannot open the key of epoch %d: %v", epoch, err)
	}
	if _, err := domainshare.OpenSeal(t.Context(), openWith(held), shareID, root, epoch, sealed); err == nil {
		t.Errorf("the device the reader revoked opens the key of epoch %d", epoch)
	}
}

func TestRefreshReader_RefusesACodeThatIsNotNewerThanTheChainOfTheShare(t *testing.T) {
	w := newWorld(t)
	w.shareFolder(t)
	w.addReader(t, w.reader, "Anna")
	code, err := w.reader.GetReaderCode(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	err = usecase.NewRefreshReader(w.owner.ShareDeps()).Execute(t.Context(), owned, folder, code.Code)

	if !errors.Is(err, domainshare.ErrStaleReaderCode) {
		t.Errorf("error = %v, want ErrStaleReaderCode for the code the reader already gave", err)
	}
}

func TestRefreshReader_RefusesAReaderTheFolderDoesNotHave(t *testing.T) {
	w := newWorld(t)
	w.shareFolder(t)
	code, err := w.reader.GetReaderCode(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	err = usecase.NewRefreshReader(w.owner.ShareDeps()).Execute(t.Context(), owned, folder, code.Code)

	if !errors.Is(err, domainshare.ErrUnknownReader) {
		t.Errorf("error = %v, want ErrUnknownReader", err)
	}
}
