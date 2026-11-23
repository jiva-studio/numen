package fakeserver_test

import (
	"bytes"
	"crypto/rand"
	"net/http"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
)

func TestAppendCatalogEntry_CommitsTheFirstEntryAndGetCatalogReturnsIt(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := newID(16)

	response := e.appendEntry(vault, true, "first")

	entries, err := e.catalog.GetCatalog(t.Context(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if response.GetHead().GetVersion() != 1 || len(entries) != 1 || entries[0].GetVersion() != 1 ||
		!bytes.Equal(entries[0].GetVaultId(), vault) || string(entries[0].GetEncBody()) != "first" {
		t.Errorf("head = %v, entries = %v, want the first entry of the vault", response.GetHead(), entries)
	}
}

func TestAppendCatalogEntry_ChainsEntriesByVersionAndPreviousHash(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	e.appendEntry(vault, false, "renamed")
	e.appendEntry(vault, false, "renamed again")

	entries, err := e.catalog.GetCatalog(t.Context(), 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 3 {
		t.Fatalf("entries = %d, want 3", len(entries))
	}
	for i := 1; i < len(entries); i++ {
		id := syncprotocol.GetCatalogEntryID(entries[i-1])
		if entries[i].GetVersion() != uint64(i+1) || !bytes.Equal(entries[i].GetPrevHash(), id[:]) {
			t.Errorf("entry %d = version %d, previous %x; want version %d and the id of the entry before", i, entries[i].GetVersion(), entries[i].GetPrevHash(), i+1)
		}
	}
}

func TestAppendCatalogEntry_DoesNotCommitAnEntryThatSkipsAVersionAndAnswersWithTheHead(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	skipping := e.newCatalogEntry(vault, false, "skips")
	skipping.Version = 3
	e.signCatalogEntry(skipping)

	response, err := e.catalog.AppendCatalogEntry(t.Context(), skipping)
	if err != nil {
		t.Fatal(err)
	}

	if response.GetCommitted() || response.GetHead().GetVersion() != 1 {
		t.Errorf("committed = %v, head version = %d, want a refusal that names the head at version 1", response.GetCommitted(), response.GetHead().GetVersion())
	}
}

func TestAppendCatalogEntry_DoesNotCommitAnEntryWithAnotherPreviousHash(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	forked := e.newCatalogEntry(vault, false, "forks")
	forked.PrevHash = bytes.Repeat([]byte{0x42}, 32)
	e.signCatalogEntry(forked)

	response, err := e.catalog.AppendCatalogEntry(t.Context(), forked)
	if err != nil {
		t.Fatal(err)
	}

	if response.GetCommitted() || response.GetHead().GetVersion() != 1 {
		t.Errorf("committed = %v, head version = %d, want a refusal that names the head at version 1", response.GetCommitted(), response.GetHead().GetVersion())
	}
	entries, err := e.catalog.GetCatalog(t.Context(), 0)
	if err != nil || len(entries) != 1 {
		t.Errorf("entries = %d, error = %v, want only the first entry", len(entries), err)
	}
}

func TestAppendCatalogEntry_TheLoserOfTwoWritersAtOneVersionGetsTheWinnerAsHead(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	winner := e.newCatalogEntry(vault, false, "winner")
	loser := e.newCatalogEntry(vault, false, "loser")

	first, err := e.catalog.AppendCatalogEntry(t.Context(), winner)
	if err != nil {
		t.Fatal(err)
	}
	second, err := e.catalog.AppendCatalogEntry(t.Context(), loser)
	if err != nil {
		t.Fatal(err)
	}

	if !first.GetCommitted() || second.GetCommitted() || string(second.GetHead().GetEncBody()) != "winner" {
		t.Errorf("first committed = %v, second committed = %v with head %q, want the winner as the head", first.GetCommitted(), second.GetCommitted(), second.GetHead().GetEncBody())
	}
}

func TestAppendCatalogEntry_RefusesAnEntryWhoseSignatureDoesNotMatchItsBytes(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	entry := e.newCatalogEntry(newID(16), true, "signed body")
	entry.EncBody = []byte("changed after signing")

	_, err := e.catalog.AppendCatalogEntry(t.Context(), entry)

	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("error = %v, want invalid argument", err)
	}
}

func TestAppendCatalogEntry_RefusesAnEntryOfAnotherPluginDomain(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	entry := e.newCatalogEntry(newID(16), true, "body")
	signature := e.device.SignForPlugin("md.numen.tools.link", syncprotocol.GetCatalogEntryMessage(entry))
	entry.Signature = signature[:]

	_, err := e.catalog.AppendCatalogEntry(t.Context(), entry)

	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("error = %v, want invalid argument", err)
	}
}

func TestAppendCatalogEntry_RefusesAnEntryOfADeviceThatIsNotInTheChainOfTheAccount(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	stranger, err := identity.NewDeviceKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	entry := e.newCatalogEntry(newID(16), true, "body")
	entry.AuthorDeviceSignKey = stranger.Public().Sign
	signature := stranger.SignForPlugin(syncprotocol.SignInPluginID, syncprotocol.GetCatalogEntryMessage(entry))
	entry.Signature = signature[:]

	_, got := e.catalog.AppendCatalogEntry(t.Context(), entry)

	if connect.CodeOf(got) != connect.CodeInvalidArgument {
		t.Errorf("error = %v, want invalid argument", got)
	}
}

func TestAppendCatalogEntry_RefusesAnEntryWithAShapeThatIsNotTheProtocol(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	entry := e.newCatalogEntry(newID(15), true, "body")
	e.signCatalogEntry(entry)

	_, err := e.catalog.AppendCatalogEntry(t.Context(), entry)

	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("error = %v, want invalid argument", err)
	}
}

func TestAppendCatalogEntry_RefusesAVaultThatExistsAndAnEntryForAVaultThatDoesNot(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	again := e.newCatalogEntry(vault, true, "again")
	unknown := e.newCatalogEntry(newID(16), false, "rename")

	_, existsErr := e.catalog.AppendCatalogEntry(t.Context(), again)
	_, unknownErr := e.catalog.AppendCatalogEntry(t.Context(), unknown)

	if connect.CodeOf(existsErr) != connect.CodeAlreadyExists {
		t.Errorf("an existing vault: error = %v, want already exists", existsErr)
	}
	if connect.CodeOf(unknownErr) != connect.CodeNotFound {
		t.Errorf("an unknown vault: error = %v, want not found", unknownErr)
	}
}

func TestACatalogCallWithoutASessionIsUnauthenticated(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	anonymous := syncclient.NewCatalogClient(e.address, http.DefaultClient, func() string { return "not-a-session" })

	_, appendErr := anonymous.AppendCatalogEntry(t.Context(), e.newCatalogEntry(newID(16), true, "body"))
	_, readErr := anonymous.GetCatalog(t.Context(), 0)

	if appendErr == nil || readErr == nil || !isUnauthenticated(appendErr) || !isUnauthenticated(readErr) {
		t.Errorf("append = %v, read = %v, want both unauthenticated", appendErr, readErr)
	}
}

func TestGetCatalogReturnsOnlyTheEntriesOfTheAccountOfTheSession(t *testing.T) {
	t.Parallel()
	_, address := newServer(t)
	first := newSecondAccountOn(t, address)
	second := newSecondAccountOn(t, address)
	first.newVault()
	second.newVault()
	second.newVault()

	firstEntries, err := first.catalog.GetCatalog(t.Context(), 0)
	if err != nil {
		t.Fatal(err)
	}
	secondEntries, err := second.catalog.GetCatalog(t.Context(), 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(firstEntries) != 1 || len(secondEntries) != 2 {
		t.Errorf("entries = %d and %d, want 1 and 2", len(firstEntries), len(secondEntries))
	}
}

func TestGetCatalogReadsMoreEntriesThanOnePageContains(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	vault := e.newVault()
	const renames = 450
	for range renames {
		e.appendEntry(vault, false, "name")
	}

	entries, err := e.catalog.GetCatalog(t.Context(), 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != renames+1 {
		t.Fatalf("entries = %d, want %d", len(entries), renames+1)
	}
	for i, entry := range entries {
		if entry.GetVersion() != uint64(i+1) {
			t.Fatalf("entry %d has version %d", i, entry.GetVersion())
		}
	}
}
