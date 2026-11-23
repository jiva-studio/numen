package syncprotocol_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

const (
	kindCreate = syncv1.ShareLogKind_SHARE_LOG_KIND_CREATE
	kindAdd    = syncv1.ShareLogKind_SHARE_LOG_KIND_ADD_READER
	kindRemove = syncv1.ShareLogKind_SHARE_LOG_KIND_REMOVE_READER
	kindRotate = syncv1.ShareLogKind_SHARE_LOG_KIND_ROTATE
	kindUpdate = syncv1.ShareLogKind_SHARE_LOG_KIND_UPDATE
	kindClose  = syncv1.ShareLogKind_SHARE_LOG_KIND_CLOSE
)

func newSeal(b byte) *syncv1.ShareSeal {
	return &syncv1.ShareSeal{ReaderRootSignKey: bytes.Repeat([]byte{b}, 32), Sealed: bytes.Repeat([]byte{b}, 84)}
}

func newEntryOfKind(kind syncv1.ShareLogKind) *syncv1.ShareLogEntry {
	entry := &syncv1.ShareLogEntry{
		ShareRootId:         bytes.Repeat([]byte{1}, 16),
		Version:             4,
		PrevHash:            bytes.Repeat([]byte{2}, 32),
		Kind:                kind,
		EncOwnerBody:        []byte("owner body"),
		EncShareBody:        []byte("share body"),
		AuthorDeviceSignKey: bytes.Repeat([]byte{5}, 32),
		Signature:           bytes.Repeat([]byte{6}, 64),
	}
	switch kind {
	case kindCreate:
		entry.Version, entry.PrevHash, entry.VaultId = 1, nil, bytes.Repeat([]byte{3}, 16)
	case kindAdd:
		entry.SubjectRootSignKey, entry.Seals = bytes.Repeat([]byte{7}, 32), []*syncv1.ShareSeal{newSeal(7)}
	case kindRemove:
		entry.Epoch, entry.SubjectRootSignKey, entry.PrevKeySeal = 1, bytes.Repeat([]byte{7}, 32), []byte("previous key")
		entry.Seals = []*syncv1.ShareSeal{newSeal(8)}
	case kindRotate:
		entry.Epoch, entry.PrevKeySeal, entry.Seals = 1, []byte("previous key"), []*syncv1.ShareSeal{newSeal(8)}
	case kindUpdate, kindClose:
	}
	return entry
}

func TestCheckShareLogEntry_AcceptsAWellFormedEntryOfEveryKind(t *testing.T) {
	t.Parallel()
	for _, kind := range []syncv1.ShareLogKind{kindCreate, kindAdd, kindRemove, kindRotate, kindUpdate, kindClose} {
		if err := syncprotocol.CheckShareLogEntry(newEntryOfKind(kind)); err != nil {
			t.Errorf("%v: error = %v", kind, err)
		}
	}
}

func TestCheckShareLogEntry_AcceptsTheLimits(t *testing.T) {
	t.Parallel()
	rotate := newEntryOfKind(kindRotate)
	rotate.Seals = nil
	for i := range syncprotocol.MaxShareSeals {
		seal := newSeal(1)
		seal.ReaderRootSignKey = append([]byte{byte(i), byte(i >> 8)}, seal.GetReaderRootSignKey()[2:]...)
		rotate.Seals = append(rotate.Seals, seal)
	}
	rotate.EncOwnerBody = make([]byte, syncprotocol.MaxEncSize)
	rotate.EncShareBody = make([]byte, syncprotocol.MaxEncShareBody)
	rotate.Seals[0].Sealed = make([]byte, 256)

	if err := syncprotocol.CheckShareLogEntry(rotate); err != nil {
		t.Errorf("error = %v", err)
	}
}

func TestCheckShareLogEntry_RefusesAShapeThatIsNotTheProtocol(t *testing.T) {
	t.Parallel()
	tooManySeals := newEntryOfKind(kindRotate)
	for range syncprotocol.MaxShareSeals + 1 {
		tooManySeals.Seals = append(tooManySeals.Seals, newSeal(1))
	}
	for name, change := range map[string]func() *syncv1.ShareLogEntry{
		"nil entry":        func() *syncv1.ShareLogEntry { return nil },
		"unspecified kind": func() *syncv1.ShareLogEntry { e := newEntryOfKind(kindUpdate); e.Kind = 0; return e },
		"short share root": func() *syncv1.ShareLogEntry {
			e := newEntryOfKind(kindUpdate)
			e.ShareRootId = e.GetShareRootId()[:15]
			return e
		},
		"version zero": func() *syncv1.ShareLogEntry { e := newEntryOfKind(kindUpdate); e.Version = 0; return e },
		"short author key": func() *syncv1.ShareLogEntry {
			e := newEntryOfKind(kindUpdate)
			e.AuthorDeviceSignKey = e.GetAuthorDeviceSignKey()[:31]
			return e
		},
		"short signature": func() *syncv1.ShareLogEntry {
			e := newEntryOfKind(kindUpdate)
			e.Signature = e.GetSignature()[:63]
			return e
		},
		"create without vault":  func() *syncv1.ShareLogEntry { e := newEntryOfKind(kindCreate); e.VaultId = nil; return e },
		"create at version two": func() *syncv1.ShareLogEntry { e := newEntryOfKind(kindCreate); e.Version = 2; return e },
		"create with previous": func() *syncv1.ShareLogEntry {
			e := newEntryOfKind(kindCreate)
			e.PrevHash = bytes.Repeat([]byte{2}, 32)
			return e
		},
		"create at epoch one": func() *syncv1.ShareLogEntry { e := newEntryOfKind(kindCreate); e.Epoch = 1; return e },
		"create with a seal": func() *syncv1.ShareLogEntry {
			e := newEntryOfKind(kindCreate)
			e.Seals = []*syncv1.ShareSeal{newSeal(7)}
			return e
		},
		"later entry no prev": func() *syncv1.ShareLogEntry { e := newEntryOfKind(kindUpdate); e.PrevHash = nil; return e },
		"add without subject": func() *syncv1.ShareLogEntry { e := newEntryOfKind(kindAdd); e.SubjectRootSignKey = nil; return e },
		"add without seal":    func() *syncv1.ShareLogEntry { e := newEntryOfKind(kindAdd); e.Seals = nil; return e },
		"add with two seals": func() *syncv1.ShareLogEntry {
			e := newEntryOfKind(kindAdd)
			e.Seals = append(e.Seals, newSeal(8))
			return e
		},
		"add seal of another": func() *syncv1.ShareLogEntry {
			e := newEntryOfKind(kindAdd)
			e.Seals = []*syncv1.ShareSeal{newSeal(8)}
			return e
		},
		"remove without subject":  func() *syncv1.ShareLogEntry { e := newEntryOfKind(kindRemove); e.SubjectRootSignKey = nil; return e },
		"remove without prev key": func() *syncv1.ShareLogEntry { e := newEntryOfKind(kindRemove); e.PrevKeySeal = nil; return e },
		"remove at epoch zero":    func() *syncv1.ShareLogEntry { e := newEntryOfKind(kindRemove); e.Epoch = 0; return e },
		"rotate without prev key": func() *syncv1.ShareLogEntry { e := newEntryOfKind(kindRotate); e.PrevKeySeal = nil; return e },
		"rotate with subject": func() *syncv1.ShareLogEntry {
			e := newEntryOfKind(kindRotate)
			e.SubjectRootSignKey = bytes.Repeat([]byte{7}, 32)
			return e
		},
		"update with subject": func() *syncv1.ShareLogEntry {
			e := newEntryOfKind(kindUpdate)
			e.SubjectRootSignKey = bytes.Repeat([]byte{7}, 32)
			return e
		},
		"update with a seal": func() *syncv1.ShareLogEntry {
			e := newEntryOfKind(kindUpdate)
			e.Seals = []*syncv1.ShareSeal{newSeal(7)}
			return e
		},
		"close with a seal": func() *syncv1.ShareLogEntry {
			e := newEntryOfKind(kindClose)
			e.Seals = []*syncv1.ShareSeal{newSeal(7)}
			return e
		},
		"close with subject": func() *syncv1.ShareLogEntry {
			e := newEntryOfKind(kindClose)
			e.SubjectRootSignKey = bytes.Repeat([]byte{7}, 32)
			return e
		},
		"too many seals": func() *syncv1.ShareLogEntry { return tooManySeals },
		"long owner body": func() *syncv1.ShareLogEntry {
			e := newEntryOfKind(kindUpdate)
			e.EncOwnerBody = make([]byte, syncprotocol.MaxEncSize+1)
			return e
		},
		"long share body": func() *syncv1.ShareLogEntry {
			e := newEntryOfKind(kindUpdate)
			e.EncShareBody = make([]byte, syncprotocol.MaxEncShareBody+1)
			return e
		},
		"long sealed": func() *syncv1.ShareLogEntry {
			e := newEntryOfKind(kindRotate)
			e.Seals[0].Sealed = make([]byte, 257)
			return e
		},
		"short seal key": func() *syncv1.ShareLogEntry {
			e := newEntryOfKind(kindRotate)
			e.Seals[0].ReaderRootSignKey = make([]byte, 31)
			return e
		},
	} {
		if err := syncprotocol.CheckShareLogEntry(change()); !errors.Is(err, syncprotocol.ErrInvalidShareLogEntry) {
			t.Errorf("%s: error = %v, want ErrInvalidShareLogEntry", name, err)
		}
	}
}

func newShareChange(left bool) *syncv1.ShareChange {
	revision := newRevision()
	revision.ShareRootId = bytes.Repeat([]byte{1}, 16)
	change := &syncv1.ShareChange{
		Change:      &syncv1.StoredRevision{Revision: revision, RevId: bytes.Repeat([]byte{9}, 32), Seq: 4},
		EncMetaHash: bytes.Repeat([]byte{3}, 32),
	}
	if left {
		change.Left, revision.EncMeta = true, nil
	}
	return change
}

func TestCheckShareRevision_AcceptsAHeadAndAStrippedHeadOfAFileThatLeft(t *testing.T) {
	t.Parallel()
	for name, change := range map[string]*syncv1.ShareChange{"head": newShareChange(false), "left": newShareChange(true)} {
		if err := syncprotocol.CheckShareRevision(change); err != nil {
			t.Errorf("%s: error = %v", name, err)
		}
	}
}

func TestCheckShareRevision_RefusesWhatCheckRevisionRefusesAndAMissingHash(t *testing.T) {
	t.Parallel()
	for name, change := range map[string]func() *syncv1.ShareChange{
		"nil":             func() *syncv1.ShareChange { return nil },
		"no revision":     func() *syncv1.ShareChange { c := newShareChange(false); c.Change = nil; return c },
		"short blob hash": func() *syncv1.ShareChange { c := newShareChange(false); c.Change.Revision.BlobHash = nil; return c },
		"no meta hash":    func() *syncv1.ShareChange { c := newShareChange(false); c.EncMetaHash = nil; return c },
		"short meta hash": func() *syncv1.ShareChange {
			c := newShareChange(true)
			c.EncMetaHash = c.GetEncMetaHash()[:31]
			return c
		},
		"empty meta not left": func() *syncv1.ShareChange { c := newShareChange(false); c.Change.Revision.EncMeta = nil; return c },
		"short signature":     func() *syncv1.ShareChange { c := newShareChange(true); c.Change.Revision.Signature = nil; return c },
	} {
		if err := syncprotocol.CheckShareRevision(change()); !errors.Is(err, syncprotocol.ErrInvalidRevision) {
			t.Errorf("%s: error = %v, want ErrInvalidRevision", name, err)
		}
	}
}
