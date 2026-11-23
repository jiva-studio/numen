package vaultlink_test

import (
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/trust"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
)

// team is an account of a laptop and a phone whose chain the laptop revoked the phone from.
type team struct {
	laptop, phone identity.DeviceKey
	before        identity.Chain
	after         identity.Chain
}

func newTeam(t *testing.T) team {
	t.Helper()
	words, err := identity.NewRecoveryWords(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	recovery, err := words.DeriveRecoveryKey()
	if err != nil {
		t.Fatal(err)
	}
	keys, err := identity.NewAccountKeys(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	at := time.Unix(1_700_000_000, 0)
	chain, err := identity.NewChain(recovery, keys, at)
	if err != nil {
		t.Fatal(err)
	}
	var devices [2]identity.DeviceKey
	for i := range devices {
		if devices[i], err = identity.NewDeviceKey(rand.Reader); err != nil {
			t.Fatal(err)
		}
		if chain, err = chain.AppendDevice(recovery, devices[i].Public(), "Device", at); err != nil {
			t.Fatal(err)
		}
	}
	after, _, err := chain.RevokeDevice(devices[0], keys, devices[1].Public().Sign, identity.Lost, rand.Reader, at.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	return team{laptop: devices[0], phone: devices[1], before: chain, after: after}
}

func (m team) authorsAfter(t *testing.T) trust.Authors {
	t.Helper()
	authors, err := trust.NewAuthors(m.before.Bytes(), m.after.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	return authors
}

// log makes the catalog entries of the authors in order, each linked to the one before.
func log(authors ...identity.DeviceKey) []*syncv1.CatalogEntry {
	var entries []*syncv1.CatalogEntry
	var previous []byte
	for i, author := range authors {
		entry := &syncv1.CatalogEntry{
			Version: uint64(i) + 1, PrevHash: previous, VaultId: make([]byte, 16), EncBody: []byte("body"), AuthorDeviceSignKey: author.Public().Sign,
		}
		signature := author.SignForPlugin(syncprotocol.SignInPluginID, syncprotocol.GetCatalogEntryMessage(entry))
		entry.Signature = signature[:]
		id := syncprotocol.GetCatalogEntryID(entry)
		previous = id[:]
		entries = append(entries, entry)
	}
	return entries
}

func TestCheckEntries_AcceptsEntriesOfARevokedDeviceThatADeviceOfTheAccountVouchedForLater(t *testing.T) {
	t.Parallel()
	m := newTeam(t)

	err := vaultlink.CheckEntries(log(m.phone, m.laptop), m.authorsAfter(t))

	if err != nil {
		t.Errorf("CheckEntries() = %v, want the entry of the phone accepted because the laptop wrote after it", err)
	}
}

func TestCheckEntries_RefusesAnEntryOfARevokedDeviceAfterTheLastEntryOfATrustedOne(t *testing.T) {
	t.Parallel()
	m := newTeam(t)

	err := vaultlink.CheckEntries(log(m.laptop, m.phone), m.authorsAfter(t))

	if !errors.Is(err, vaultlink.ErrInvalidCatalog) {
		t.Errorf("CheckEntries() = %v, want ErrInvalidCatalog for what the phone appended last", err)
	}
}

func TestCheckEntries_RefusesALogOfRevokedDevicesOnly(t *testing.T) {
	t.Parallel()
	m := newTeam(t)

	if err := vaultlink.CheckEntries(log(m.phone), m.authorsAfter(t)); !errors.Is(err, vaultlink.ErrInvalidCatalog) {
		t.Errorf("CheckEntries() = %v, want ErrInvalidCatalog", err)
	}
}

func TestCheckEntries_RefusesAnEntryOfAKeyThatNeverWasADeviceOfTheAccount(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	stranger, err := identity.NewDeviceKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	if err := vaultlink.CheckEntries(log(stranger, m.laptop), m.authorsAfter(t)); !errors.Is(err, vaultlink.ErrInvalidCatalog) {
		t.Errorf("CheckEntries() = %v, want ErrInvalidCatalog for the entry of a stranger", err)
	}
}
