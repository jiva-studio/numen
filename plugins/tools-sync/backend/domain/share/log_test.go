package share_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/trust"
)

func getLogShareID() []byte { return bytes.Repeat([]byte{7}, 16) }

type person struct {
	keys   identity.AccountKeys
	device identity.DeviceKey
	chain  identity.Chain
}

func newPerson(t *testing.T) person {
	t.Helper()
	words, err := identity.NewRecoveryWords(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	root, err := words.DeriveRecoveryKey()
	if err != nil {
		t.Fatal(err)
	}
	keys, err := identity.NewAccountKeys(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	device, err := identity.NewDeviceKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, time.November, 23, 12, 0, 0, 0, time.UTC)
	chain, err := identity.NewChain(root, keys, at)
	if err != nil {
		t.Fatal(err)
	}
	if chain, err = chain.AppendDevice(root, device.Public(), "laptop", at); err != nil {
		t.Fatal(err)
	}
	return person{keys: keys, device: device, chain: chain}
}

// world is a share log of one owner with one reader, entry by entry as the owner makes them.
type world struct {
	owner   person
	reader  person
	entries []*syncv1.ShareLogEntry
	keys    [][32]byte
}

func newWorld(t *testing.T) *world {
	t.Helper()
	w := &world{owner: newPerson(t), reader: newPerson(t)}
	w.keys = append(w.keys, newKey(1))
	w.append(t, &syncv1.ShareLogEntry{Kind: syncv1.ShareLogKind_SHARE_LOG_KIND_CREATE, VaultId: bytes.Repeat([]byte{5}, 16)})
	w.append(t, &syncv1.ShareLogEntry{
		Kind: syncv1.ShareLogKind_SHARE_LOG_KIND_ADD_READER, SubjectRootSignKey: w.reader.chain.Root().Sign,
		Seals: []*syncv1.ShareSeal{w.seal(t, 0)},
	})
	w.keys = append(w.keys, newKey(2))
	previous, err := share.SealPreviousKey(rand.Reader, w.keys[1], w.keys[0], getLogShareID(), 1)
	if err != nil {
		t.Fatal(err)
	}
	w.append(t, &syncv1.ShareLogEntry{
		Kind: syncv1.ShareLogKind_SHARE_LOG_KIND_ROTATE, Epoch: 1, PrevKeySeal: previous, Seals: []*syncv1.ShareSeal{w.seal(t, 1)},
	})
	return w
}

func (w *world) seal(t *testing.T, epoch uint32) *syncv1.ShareSeal {
	t.Helper()
	root := w.reader.chain.Root().Sign
	sealed, err := share.SealToReader(rand.Reader, w.reader.chain.AccountBoxKey(), getLogShareID(), root, epoch, w.keys[epoch])
	if err != nil {
		t.Fatal(err)
	}
	return &syncv1.ShareSeal{ReaderRootSignKey: root, Sealed: sealed}
}

func (w *world) append(t *testing.T, entry *syncv1.ShareLogEntry) {
	t.Helper()
	entry.ShareRootId, entry.Version = getLogShareID(), uint64(len(w.entries))+1
	if len(w.entries) > 0 {
		id := syncprotocol.GetShareLogEntryID(w.entries[len(w.entries)-1])
		entry.PrevHash = id[:]
	}
	key := w.keys[entry.GetEpoch()]
	body, err := share.SealBody(rand.Reader, key, getLogShareID(), entry.GetEpoch(), share.Body{Name: "Food"})
	if err != nil {
		t.Fatal(err)
	}
	entry.EncOwnerBody, entry.EncShareBody = []byte("owner body"), body
	w.sign(entry, w.owner.device)
	w.entries = append(w.entries, entry)
}

func (w *world) sign(entry *syncv1.ShareLogEntry, device identity.DeviceKey) {
	entry.AuthorDeviceSignKey = device.Public().Sign
	signature := device.SignForPlugin(syncprotocol.SignInPluginID, syncprotocol.GetShareLogEntryMessage(entry))
	entry.Signature = signature[:]
}

func (w *world) rules(t *testing.T) share.Rules {
	t.Helper()
	authors, err := trust.NewAuthors(w.owner.chain.Bytes(), w.owner.chain.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	open := func(_ context.Context, boxContext, sealed []byte) ([]byte, error) {
		return w.reader.keys.OpenSealedBox(syncprotocol.SignInPluginID, boxContext, sealed)
	}
	return share.Rules{ShareRootID: getLogShareID(), Reader: w.reader.chain.Root().Sign, Authors: authors, Open: open}
}

func (w *world) follow(t *testing.T, entries []*syncv1.ShareLogEntry) (share.Position, error) {
	t.Helper()
	follower := share.NewFollower(w.rules(t), share.Position{})
	err := follower.Follow(t.Context(), entries)
	return follower.GetPosition(), err
}

func clone(entries []*syncv1.ShareLogEntry) []*syncv1.ShareLogEntry {
	copied := make([]*syncv1.ShareLogEntry, len(entries))
	for i, entry := range entries {
		copied[i] = &syncv1.ShareLogEntry{
			ShareRootId: entry.GetShareRootId(), Version: entry.GetVersion(), PrevHash: entry.GetPrevHash(), Kind: entry.GetKind(),
			VaultId: entry.GetVaultId(), Epoch: entry.GetEpoch(), SubjectRootSignKey: entry.GetSubjectRootSignKey(),
			EncOwnerBody: entry.GetEncOwnerBody(), EncShareBody: entry.GetEncShareBody(), Seals: entry.GetSeals(),
			PrevKeySeal: entry.GetPrevKeySeal(), AuthorDeviceSignKey: entry.GetAuthorDeviceSignKey(), Signature: entry.GetSignature(),
		}
	}
	return copied
}

func TestFollower_ReadsEveryEpochKeyAndShareName(t *testing.T) {
	w := newWorld(t)

	position, err := w.follow(t, w.entries)

	if err != nil {
		t.Fatal(err)
	}
	if position.Version != 3 || position.Epoch != 1 || !position.IsActive || position.IsEnded || position.Name != "Food" {
		t.Errorf("position = %+v, want version 3, epoch 1, active and the name Food", position)
	}
	for epoch, want := range w.keys {
		if got, ok := position.Keys.Get(uint32(epoch)); !ok || got != want {
			t.Errorf("key of epoch %d = %x (%v), want %x", epoch, got, ok, want)
		}
	}
}

func newLateWorld(t *testing.T) *world {
	t.Helper()
	w := &world{owner: newPerson(t), reader: newPerson(t)}
	w.keys = [][32]byte{newKey(1), newKey(2)}
	w.append(t, &syncv1.ShareLogEntry{Kind: syncv1.ShareLogKind_SHARE_LOG_KIND_CREATE, VaultId: bytes.Repeat([]byte{5}, 16)})
	previous, err := share.SealPreviousKey(rand.Reader, w.keys[1], w.keys[0], getLogShareID(), 1)
	if err != nil {
		t.Fatal(err)
	}
	w.append(t, &syncv1.ShareLogEntry{Kind: syncv1.ShareLogKind_SHARE_LOG_KIND_ROTATE, Epoch: 1, PrevKeySeal: previous})
	w.append(t, &syncv1.ShareLogEntry{
		Kind: syncv1.ShareLogKind_SHARE_LOG_KIND_ADD_READER, Epoch: 1, SubjectRootSignKey: w.reader.chain.Root().Sign,
		Seals: []*syncv1.ShareSeal{w.seal(t, 1)},
	})
	return w
}

func TestFollower_OpensTheEarlierEpochsOfAReaderThatJoinedLaterFromThePreviousKeySeals(t *testing.T) {
	w := newLateWorld(t)

	position, err := w.follow(t, w.entries)

	if err != nil {
		t.Fatal(err)
	}
	for epoch, want := range w.keys {
		if got, ok := position.Keys.Get(uint32(epoch)); !ok || got != want {
			t.Errorf("key of epoch %d = %x (%v), want %x", epoch, got, ok, want)
		}
	}
}

func mustPrev(entry *syncv1.ShareLogEntry) []byte {
	id := syncprotocol.GetShareLogEntryID(entry)
	return id[:]
}

func TestFollower_EndsTheShareAtTheRemovalAndFollowsTheLogOn(t *testing.T) {
	w := newWorld(t)
	w.keys = append(w.keys, newKey(3))
	previous, err := share.SealPreviousKey(rand.Reader, w.keys[2], w.keys[1], getLogShareID(), 2)
	if err != nil {
		t.Fatal(err)
	}
	w.append(t, &syncv1.ShareLogEntry{
		Kind: syncv1.ShareLogKind_SHARE_LOG_KIND_REMOVE_READER, Epoch: 2, SubjectRootSignKey: w.reader.chain.Root().Sign, PrevKeySeal: previous,
	})
	w.append(t, &syncv1.ShareLogEntry{Kind: syncv1.ShareLogKind_SHARE_LOG_KIND_UPDATE, Epoch: 2})

	position, err := w.follow(t, w.entries)

	if err != nil {
		t.Fatal(err)
	}
	if !position.IsEnded || position.IsActive || position.Version != 5 {
		t.Errorf("position = %+v, want ended, at version 5 of the log that goes on after the removal", position)
	}
	if _, ok := position.Keys.Get(2); ok {
		t.Error("the removed reader holds the key of the epoch that started with its removal")
	}
}

func TestFollower_RefusesALogTheOwnerDidNotSignAsItIs(t *testing.T) {
	stranger := newPerson(t)
	for name, damage := range map[string]func(w *world, entries []*syncv1.ShareLogEntry) []*syncv1.ShareLogEntry{
		"a gap in the versions": func(_ *world, entries []*syncv1.ShareLogEntry) []*syncv1.ShareLogEntry {
			return []*syncv1.ShareLogEntry{entries[0], entries[2]}
		},
		"a previous hash that is not the entry before": func(w *world, entries []*syncv1.ShareLogEntry) []*syncv1.ShareLogEntry {
			entries[1].PrevHash = bytes.Repeat([]byte{9}, 32)
			w.sign(entries[1], w.owner.device)
			return entries
		},
		"a bad signature": func(_ *world, entries []*syncv1.ShareLogEntry) []*syncv1.ShareLogEntry {
			entries[1].Signature = append([]byte(nil), entries[1].GetSignature()...)
			entries[1].Signature[0] ^= 1
			return entries
		},
		"an author outside the chain of the owner": func(w *world, entries []*syncv1.ShareLogEntry) []*syncv1.ShareLogEntry {
			w.sign(entries[1], stranger.device)
			return entries
		},
		"another share": func(w *world, entries []*syncv1.ShareLogEntry) []*syncv1.ShareLogEntry {
			entries[1].ShareRootId = bytes.Repeat([]byte{8}, 16)
			w.sign(entries[1], w.owner.device)
			return entries
		},
		"an epoch that goes down": func(w *world, entries []*syncv1.ShareLogEntry) []*syncv1.ShareLogEntry {
			down := &syncv1.ShareLogEntry{
				ShareRootId: getLogShareID(), Version: 4, PrevHash: mustPrev(entries[2]), Kind: syncv1.ShareLogKind_SHARE_LOG_KIND_UPDATE, Epoch: 0,
				EncOwnerBody: []byte("owner body"), EncShareBody: entries[1].GetEncShareBody(),
			}
			w.sign(down, w.owner.device)
			return append(entries, down)
		},
		"a seal for another reader": func(w *world, entries []*syncv1.ShareLogEntry) []*syncv1.ShareLogEntry {
			entries[2].Seals = []*syncv1.ShareSeal{{ReaderRootSignKey: stranger.chain.Root().Sign, Sealed: entries[2].GetSeals()[0].GetSealed()}}
			w.sign(entries[2], w.owner.device)
			return entries
		},
		"a seal of another epoch": func(w *world, entries []*syncv1.ShareLogEntry) []*syncv1.ShareLogEntry {
			entries[2].Seals = []*syncv1.ShareSeal{w.seal(t, 0)}
			w.sign(entries[2], w.owner.device)
			return entries
		},
		"a creation after the first entry": func(w *world, entries []*syncv1.ShareLogEntry) []*syncv1.ShareLogEntry {
			entries[1].Kind, entries[1].SubjectRootSignKey, entries[1].Seals = syncv1.ShareLogKind_SHARE_LOG_KIND_CREATE, nil, nil
			w.sign(entries[1], w.owner.device)
			return entries
		},
	} {
		w := newWorld(t)

		_, err := w.follow(t, damage(w, clone(w.entries)))

		if !errors.Is(err, share.ErrDamagedLog) {
			t.Errorf("%s: error = %v, want ErrDamagedLog", name, err)
		}
	}
}

func TestFollower_RefusesAnOwnerChainOfAnotherRootBeforeAnyEntry(t *testing.T) {
	w := newWorld(t)
	stranger := newPerson(t)

	_, err := trust.NewAuthors(stranger.chain.Bytes(), w.owner.chain.Bytes())

	if err == nil {
		t.Error("the chain of another root was taken for the chain of the owner")
	}
}

func TestFollower_ContinuesFromTheEntryItVerifiedLast(t *testing.T) {
	w := newWorld(t)
	first := share.NewFollower(w.rules(t), share.Position{})
	if err := first.Follow(t.Context(), w.entries[:2]); err != nil {
		t.Fatal(err)
	}

	next := share.NewFollower(w.rules(t), first.GetPosition())
	err := next.Follow(t.Context(), w.entries[2:])

	if err != nil {
		t.Fatal(err)
	}
	if got := next.GetPosition(); got.Version != 3 || got.Epoch != 1 {
		t.Errorf("position = %+v, want version 3 and epoch 1", got)
	}
}
