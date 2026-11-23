package catalog_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/account"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/wiretest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/cloud"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/catalog"
)

type world struct {
	server *synctest.Server
	clock  *synctest.Clock
	words  []string
	first  *synctest.Computer
}

func newWorld(t *testing.T) *world {
	t.Helper()
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{})
	words := synctest.NewWords(t)
	return &world{server: server, clock: clock, words: words, first: synctest.NewFirstComputer(t, server, clock, "Test", words)}
}

func (w *world) newSecond(t *testing.T) *synctest.Computer {
	t.Helper()
	return synctest.NewSecondComputer(t, w.server, w.clock, "Copy", w.words)
}

func listRemote(t *testing.T, c *synctest.Computer) []vaultlink.RemoteVault {
	t.Helper()
	vaults, err := catalog.NewListRemoteVaults(c.CatalogDeps()).Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return vaults
}

type racingCatalog struct {
	*cloud.Cloud

	before   func()
	attempts int
}

func (r *racingCatalog) AppendCatalogEntry(ctx context.Context, entry *syncv1.CatalogEntry) (*syncv1.AppendCatalogEntryResponse, error) {
	r.attempts++
	if r.before != nil {
		r.before()
	}
	return r.Cloud.AppendCatalogEntry(ctx, entry)
}

type tamperedCatalog struct {
	*cloud.Cloud

	change func([]*syncv1.CatalogEntry) []*syncv1.CatalogEntry
}

func (r *tamperedCatalog) GetCatalog(ctx context.Context, since uint64) ([]*syncv1.CatalogEntry, error) {
	entries, err := r.Cloud.GetCatalog(ctx, since)
	if err != nil {
		return nil, err
	}
	cloned := make([]*syncv1.CatalogEntry, len(entries))
	for i, entry := range entries {
		cloned[i] = wiretest.CloneCatalogEntry(entry)
	}
	return r.change(cloned), nil
}

func TestConnectVault_AppendsTheCreatingEntryOfANewVault(t *testing.T) {
	w := newWorld(t)

	w.first.Connect(t, "Test")

	entries := w.first.ReadCatalog(t)
	if len(entries) != 1 {
		t.Fatalf("catalog has %d entries, want 1", len(entries))
	}
	entry := entries[0]
	public := w.first.Fake.Public()
	if entry.GetVersion() != 1 || len(entry.GetPrevHash()) != 0 || !entry.GetCreatesVault() || len(entry.GetVaultId()) != 16 {
		t.Errorf("entry = version %d, prev %x, creates %v, vault %x; want version 1, no prev, creating, 16 bytes",
			entry.GetVersion(), entry.GetPrevHash(), entry.GetCreatesVault(), entry.GetVaultId())
	}
	if !bytes.Equal(entry.GetAuthorDeviceSignKey(), public.DeviceSignKey) {
		t.Error("the author of the entry is not the device of this computer")
	}
	if !identity.VerifyForPlugin(entry.GetAuthorDeviceSignKey(), syncprotocol.SignInPluginID, syncprotocol.GetCatalogEntryMessage(entry), [64]byte(entry.GetSignature())) {
		t.Error("the signature of the entry does not verify under the plugin id")
	}
}

func TestConnectVault_SealsTheNameAndTheKeyInTheBody(t *testing.T) {
	w := newWorld(t)

	w.first.Connect(t, "Test")

	entry := w.first.ReadCatalog(t)[0]
	body := w.first.OpenCatalogBody(t, entry)
	if body.Name != "Test" || body.LocalID != "Test" || len(body.Key) != 32 {
		t.Errorf("body = %+v, want the name and the local id Test and a key of 32 bytes", body)
	}
	if bytes.Contains(entry.GetEncBody(), []byte("Test")) || bytes.Contains(entry.GetEncBody(), body.Key) {
		t.Error("the body on the server holds the name or the key as plain bytes")
	}
}

func TestConnectVault_SealsTheBodyForTheKeysOfTheAccountOnly(t *testing.T) {
	w := newWorld(t)
	w.first.Connect(t, "Test")
	entry := w.first.ReadCatalog(t)[0]
	stranger := synctest.NewFirstComputer(t, w.server, w.clock, "Other", synctest.NewWords(t))

	_, err := stranger.Client.OpenData(t.Context(), synctest.GetCatalogContext(entry), entry.GetEncBody())

	if !errors.Is(err, account.ErrCannotOpen) {
		t.Errorf("a computer of another account opened the body: error = %v, want ErrCannotOpen", err)
	}
}

func TestConnectVault_StoresTheLinkWithTheVaultKeySealed(t *testing.T) {
	w := newWorld(t)

	w.first.Connect(t, "Test")

	entry := w.first.ReadCatalog(t)[0]
	body := w.first.OpenCatalogBody(t, entry)
	link := w.first.Link(t, "Test")
	if link.RemoteVaultID != hex.EncodeToString(entry.GetVaultId()) || link.CatalogName != "Test" || link.ServerSeq != 0 {
		t.Errorf("link = %+v, want the remote id %x, the name Test and no server position", link, entry.GetVaultId())
	}
	if bytes.Contains(link.SealedKey, body.Key) {
		t.Error("link.json holds the vault key as plain bytes")
	}
	if key := w.first.GetVaultKey(t, "Test"); !bytes.Equal(key[:], body.Key) {
		t.Error("the sealed key of link.json opens to another key than the one of the catalog")
	}
}

func TestConnectVault_ConnectsAFolderOfASecondComputerToAnExistingVault(t *testing.T) {
	w := newWorld(t)
	w.first.Connect(t, "Test")
	second := w.newSecond(t)
	listed := listRemote(t, second)
	if len(listed) != 1 || listed[0].Name != "Test" || listed[0].IsConnected {
		t.Fatalf("remote vaults of the second computer = %+v, want Test, not connected", listed)
	}

	second.ConnectExisting(t, "Copy", listed[0].RemoteID)

	if link := second.Link(t, "Copy"); link.RemoteVaultID != w.first.Link(t, "Test").RemoteVaultID {
		t.Errorf("remote id of the copy = %s, want the one of the first computer", link.RemoteVaultID)
	}
	firstKey, secondKey := w.first.GetVaultKey(t, "Test"), second.GetVaultKey(t, "Copy")
	if firstKey != secondKey {
		t.Error("the second computer opened another vault key than the first one made")
	}
	if got := len(second.ReadCatalog(t)); got != 1 {
		t.Errorf("catalog has %d entries after the join, want 1", got)
	}
	if listed := listRemote(t, second); !listed[0].IsConnected {
		t.Error("the vault is not connected on the second computer after the join")
	}
}

func TestConnectVault_RefusesAnUnknownRemoteVault(t *testing.T) {
	w := newWorld(t)

	err := catalog.NewConnectVault(w.first.CatalogDeps()).Execute(t.Context(),
		catalog.ConnectInput{VaultID: "Test", Mode: catalog.ModeExisting, RemoteID: "00112233445566778899aabbccddeeff"})

	if !errors.Is(err, catalog.ErrUnknownRemoteVault) {
		t.Errorf("error = %v, want ErrUnknownRemoteVault", err)
	}
	if _, err := w.first.Host.Store("Test").Read(t.Context(), synctest.LinkFile); !errors.Is(err, host.ErrNotExist) {
		t.Errorf("link.json read error = %v, want none written", err)
	}
}

func TestConnectVault_RefusesACopyOfAConnectedVault(t *testing.T) {
	w := newWorld(t)
	w.first.Connect(t, "Test")
	w.first.AddVault(t, "Test copy")
	w.first.Vaults.Store().With("Test copy").WriteFile(synctest.LinkFile, w.first.ReadStoreFile(t, "Test", synctest.LinkFile))

	asNew := catalog.NewConnectVault(w.first.CatalogDeps()).Execute(t.Context(), catalog.ConnectInput{VaultID: "Test copy", Mode: catalog.ModeNew})
	asExisting := catalog.NewConnectVault(w.first.CatalogDeps()).Execute(t.Context(),
		catalog.ConnectInput{VaultID: "Test copy", Mode: catalog.ModeExisting, RemoteID: w.first.Link(t, "Test").RemoteVaultID})

	if !errors.Is(asNew, catalog.ErrCopyOfConnectedVault) || !errors.Is(asExisting, catalog.ErrCopyOfConnectedVault) {
		t.Errorf("errors = %v and %v, want ErrCopyOfConnectedVault for both", asNew, asExisting)
	}
	if got := len(w.first.ReadCatalog(t)); got != 1 {
		t.Errorf("catalog has %d entries after the refusals, want 1", got)
	}
}

func TestConnectVault_ReadsTheCatalogAgainWhenAnotherDeviceAppendedFirst(t *testing.T) {
	w := newWorld(t)
	second := w.newSecond(t)
	second.AddVault(t, "Other")
	racing := &racingCatalog{Cloud: w.first.Cloud}
	racing.before = func() {
		racing.before = nil
		second.Connect(t, "Other")
	}
	deps := w.first.CatalogDeps()
	deps.Catalog = racing

	if err := catalog.NewConnectVault(deps).Execute(t.Context(), catalog.ConnectInput{VaultID: "Test", Mode: catalog.ModeNew}); err != nil {
		t.Fatal(err)
	}

	entries := w.first.ReadCatalog(t)
	if len(entries) != 2 || entries[1].GetVersion() != 2 {
		t.Fatalf("catalog = %v, want the entry of the other device and then ours as version 2", entries)
	}
	previous := syncprotocol.GetCatalogEntryID(entries[0])
	if !bytes.Equal(entries[1].GetPrevHash(), previous[:]) {
		t.Error("the second entry does not name the first one as its previous entry")
	}
	if racing.attempts != 2 {
		t.Errorf("append attempts = %d, want 2", racing.attempts)
	}
}

func TestConnectVault_FailsAfterThreeLostRaces(t *testing.T) {
	w := newWorld(t)
	second := w.newSecond(t)
	racing := &racingCatalog{Cloud: w.first.Cloud}
	rival := 0
	racing.before = func() {
		rival++
		name := "Other " + string(rune('0'+rival))
		second.AddVault(t, name)
		second.Connect(t, domain.VaultID(name))
	}
	deps := w.first.CatalogDeps()
	deps.Catalog = racing

	err := catalog.NewConnectVault(deps).Execute(t.Context(), catalog.ConnectInput{VaultID: "Test", Mode: catalog.ModeNew})

	if !errors.Is(err, catalog.ErrCatalogBusy) {
		t.Errorf("error = %v, want ErrCatalogBusy", err)
	}
	if racing.attempts != 3 {
		t.Errorf("append attempts = %d, want 3", racing.attempts)
	}
	if _, err := w.first.Host.Store("Test").Read(t.Context(), synctest.LinkFile); !errors.Is(err, host.ErrNotExist) {
		t.Errorf("link.json read error = %v, want none written", err)
	}
}

func TestConnectVault_NeedsAnIdentity(t *testing.T) {
	w := newWorld(t)
	w.first.Fake.NoIdentity()

	err := catalog.NewConnectVault(w.first.CatalogDeps()).Execute(t.Context(), catalog.ConnectInput{VaultID: "Test", Mode: catalog.ModeNew})

	if !errors.Is(err, session.ErrNoIdentity) {
		t.Errorf("error = %v, want ErrNoIdentity", err)
	}
	if got := len(w.first.ReadCatalog(t)); got != 0 {
		t.Errorf("catalog has %d entries, want none", got)
	}
}

func TestListRemoteVaults_FoldsTheEntriesOfAVaultToItsLastName(t *testing.T) {
	w := newWorld(t)
	w.first.AddVault(t, "Work")
	w.first.Connect(t, "Test")
	w.first.Connect(t, "Work")
	if _, err := w.first.Vaults.RenameVault(t.Context(), "Test", "Notes"); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.NewRenameRemoteVault(w.first.CatalogDeps()).Execute(t.Context(), "Test"); err != nil {
		t.Fatal(err)
	}
	second := w.newSecond(t)

	listed := listRemote(t, second)

	if len(listed) != 2 || listed[0].Name != "Notes" || listed[1].Name != "Work" {
		t.Fatalf("remote vaults = %+v, want Notes and Work in the order they were made", listed)
	}
	second.ConnectExisting(t, "Copy", listed[1].RemoteID)
	if again := listRemote(t, second); again[0].IsConnected || !again[1].IsConnected {
		t.Errorf("connected = %v and %v, want only Work connected on the second computer", again[0].IsConnected, again[1].IsConnected)
	}
}

func TestListRemoteVaults_TrustsADeviceAddedAfterThisChain(t *testing.T) {
	w := newWorld(t)
	w.first.Connect(t, "Test")
	second := w.newSecond(t)
	second.AddVault(t, "Other")
	second.Connect(t, "Other")

	listed := listRemote(t, w.first)

	if len(listed) != 2 {
		t.Errorf("remote vaults = %+v, want the vault of the second computer as well", listed)
	}
}

func TestRenameRemoteVault_AppendsANonCreatingEntryWithTheNewName(t *testing.T) {
	w := newWorld(t)
	w.first.Connect(t, "Test")
	if _, err := w.first.Vaults.RenameVault(t.Context(), "Test", "Work"); err != nil {
		t.Fatal(err)
	}

	renamed, err := catalog.NewRenameRemoteVault(w.first.CatalogDeps()).Execute(t.Context(), "Test")
	if err != nil {
		t.Fatal(err)
	}

	entries := w.first.ReadCatalog(t)
	if !renamed || len(entries) != 2 || entries[1].GetCreatesVault() || !bytes.Equal(entries[1].GetVaultId(), entries[0].GetVaultId()) {
		t.Fatalf("renamed = %v, catalog = %v; want a second entry of the same vault that does not create it", renamed, entries)
	}
	first, second := w.first.OpenCatalogBody(t, entries[0]), w.first.OpenCatalogBody(t, entries[1])
	if second.Name != "Work" || !bytes.Equal(first.Key, second.Key) {
		t.Errorf("second body = %+v, want the name Work and the same key", second)
	}
	if link := w.first.Link(t, "Test"); link.CatalogName != "Work" || link.RemoteVaultID != hex.EncodeToString(entries[0].GetVaultId()) {
		t.Errorf("link = %+v, want the name Work and the same remote id", link)
	}
}

func TestRenameRemoteVault_AppendsNothingWhenTheNameIsTheSame(t *testing.T) {
	w := newWorld(t)
	w.first.Connect(t, "Test")

	renamed, err := catalog.NewRenameRemoteVault(w.first.CatalogDeps()).Execute(t.Context(), "Test")
	if err != nil {
		t.Fatal(err)
	}

	if renamed || len(w.first.ReadCatalog(t)) != 1 {
		t.Errorf("renamed = %v with %d entries, want no entry added", renamed, len(w.first.ReadCatalog(t)))
	}
}

func TestListRemoteVaults_RefusesACatalogThatIsNotValid(t *testing.T) {
	changes := map[string]func(t *testing.T, w *world, entries []*syncv1.CatalogEntry) []*syncv1.CatalogEntry{
		"a gap in the versions": func(_ *testing.T, _ *world, entries []*syncv1.CatalogEntry) []*syncv1.CatalogEntry {
			return entries[1:]
		},
		"a flipped signature": func(_ *testing.T, _ *world, entries []*syncv1.CatalogEntry) []*syncv1.CatalogEntry {
			entries[1].Signature[0] ^= 1
			return entries
		},
		"a wrong previous hash under a valid signature": func(t *testing.T, w *world, entries []*syncv1.CatalogEntry) []*syncv1.CatalogEntry {
			t.Helper()
			entries[1].PrevHash = make([]byte, 32)
			w.first.SignCatalogEntry(t, entries[1])
			return entries
		},
		"an author that is not a device of the account": func(t *testing.T, w *world, entries []*syncv1.CatalogEntry) []*syncv1.CatalogEntry {
			t.Helper()
			synctest.NewFirstComputer(t, w.server, w.clock, "Other", synctest.NewWords(t)).SignCatalogEntry(t, entries[1])
			return entries
		},
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			w := newWorld(t)
			w.first.AddVault(t, "Work")
			w.first.Connect(t, "Test")
			w.first.Connect(t, "Work")
			deps := w.first.CatalogDeps()
			deps.Catalog = &tamperedCatalog{Cloud: w.first.Cloud, change: func(entries []*syncv1.CatalogEntry) []*syncv1.CatalogEntry { return change(t, w, entries) }}

			_, err := catalog.NewListRemoteVaults(deps).Execute(t.Context())

			if !errors.Is(err, catalog.ErrInvalidCatalog) {
				t.Errorf("error = %v, want ErrInvalidCatalog", err)
			}
		})
	}
}
