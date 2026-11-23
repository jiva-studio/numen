package share

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

const sealedKeySize = 4 + keySize

// Opener opens a box sealed to the account of this device for the sync plugin.
type Opener func(ctx context.Context, boxContext, sealed []byte) ([]byte, error)

// Authors tells the signing keys of the devices of the account that wrote the log.
type Authors interface {
	IsTrusted(signKey []byte) bool
}

// Rules are what a reader checks a share log against.
type Rules struct {
	ShareRootID []byte
	Reader      []byte
	Authors     Authors
	Open        Opener
}

// Position is what a reader knows of a share log it has verified: its last entry, the keys it holds and whether the
// share still reads.
type Position struct {
	VaultID  []byte
	Version  uint64
	HeadID   [32]byte
	Epoch    uint32
	Keys     Keys
	Name     string
	IsActive bool
	IsEnded  bool
}

// Follower verifies the entries of a share log one after the other and keeps the position they lead to.
type Follower struct {
	rules    Rules
	position Position
	previous map[uint32][]byte
}

// NewFollower returns a follower that continues from the position.
func NewFollower(rules Rules, position Position) *Follower {
	return &Follower{rules: rules, position: position, previous: map[uint32][]byte{}}
}

// GetPosition returns the position after the entries followed so far.
func (f *Follower) GetPosition() Position { return f.position }

// Follow verifies the entries, which must continue the position, and takes the seals addressed to the reader. The
// entry that removes the reader ends its share; a later entry that adds the same reader again starts it anew, and the
// keys of the epochs before stay with the reader.
func (f *Follower) Follow(ctx context.Context, entries []*syncv1.ShareLogEntry) error {
	for _, entry := range entries {
		if err := f.apply(ctx, entry); err != nil {
			return err
		}
	}
	return f.walkBack()
}

func (f *Follower) apply(ctx context.Context, entry *syncv1.ShareLogEntry) error {
	if err := f.check(entry); err != nil {
		return fmt.Errorf("%w: entry %d: %w", ErrDamagedLog, entry.GetVersion(), err)
	}
	f.position.Version = entry.GetVersion()
	f.position.HeadID = syncprotocol.GetShareLogEntryID(entry)
	f.position.Epoch = entry.GetEpoch()
	if len(entry.GetPrevKeySeal()) > 0 {
		f.previous[entry.GetEpoch()] = entry.GetPrevKeySeal()
	}
	if err := f.take(ctx, entry); err != nil {
		return fmt.Errorf("%w: entry %d: %w", ErrDamagedLog, entry.GetVersion(), err)
	}
	f.readName(entry)
	return nil
}

func (f *Follower) take(ctx context.Context, entry *syncv1.ShareLogEntry) error {
	isMe := bytes.Equal(entry.GetSubjectRootSignKey(), f.rules.Reader)
	switch entry.GetKind() {
	case syncv1.ShareLogKind_SHARE_LOG_KIND_CREATE:
		f.position.VaultID = entry.GetVaultId()
	case syncv1.ShareLogKind_SHARE_LOG_KIND_ADD_READER:
		if isMe {
			f.position.IsActive, f.position.IsEnded = true, false
			return f.takeSeal(ctx, entry)
		}
	case syncv1.ShareLogKind_SHARE_LOG_KIND_REMOVE_READER:
		if isMe {
			f.end()
			return nil
		}
		return f.takeIfActive(ctx, entry)
	case syncv1.ShareLogKind_SHARE_LOG_KIND_ROTATE:
		return f.takeIfActive(ctx, entry)
	case syncv1.ShareLogKind_SHARE_LOG_KIND_CLOSE:
		if f.position.IsActive {
			f.end()
		}
	default:
	}
	return nil
}

func (f *Follower) end() {
	f.position.IsActive = false
	f.position.IsEnded = true
}

func (f *Follower) takeIfActive(ctx context.Context, entry *syncv1.ShareLogEntry) error {
	if !f.position.IsActive {
		return nil
	}
	return f.takeSeal(ctx, entry)
}

func (f *Follower) takeSeal(ctx context.Context, entry *syncv1.ShareLogEntry) error {
	for _, one := range entry.GetSeals() {
		if !bytes.Equal(one.GetReaderRootSignKey(), f.rules.Reader) {
			continue
		}
		key, err := OpenSeal(ctx, f.rules.Open, f.rules.ShareRootID, f.rules.Reader, entry.GetEpoch(), one.GetSealed())
		if err != nil {
			return err
		}
		f.position.Keys = f.position.Keys.Add(entry.GetEpoch(), key)
		return nil
	}
	return errNoSeal
}

func (f *Follower) check(entry *syncv1.ShareLogEntry) error {
	if err := syncprotocol.CheckShareLogEntry(entry); err != nil {
		return err
	}
	if entry.GetVersion() != f.position.Version+1 {
		return errWrongVersion
	}
	isFirst := entry.GetVersion() == 1
	if isFirst != (entry.GetKind() == syncv1.ShareLogKind_SHARE_LOG_KIND_CREATE) {
		return errWrongKind
	}
	if !isFirst && !bytes.Equal(entry.GetPrevHash(), f.position.HeadID[:]) {
		return errWrongPrevious
	}
	if !bytes.Equal(entry.GetShareRootId(), f.rules.ShareRootID) {
		return errWrongShare
	}
	if entry.GetEpoch() < f.position.Epoch {
		return errEpochDown
	}
	return f.checkAuthor(entry)
}

func (f *Follower) checkAuthor(entry *syncv1.ShareLogEntry) error {
	author := entry.GetAuthorDeviceSignKey()
	if !f.rules.Authors.IsTrusted(author) {
		return errUnknownAuthor
	}
	if len(entry.GetSignature()) != 64 ||
		!identity.VerifyForPlugin(author, syncprotocol.SignInPluginID, syncprotocol.GetShareLogEntryMessage(entry), [64]byte(entry.GetSignature())) {
		return errBadSignature
	}
	return nil
}

func (f *Follower) readName(entry *syncv1.ShareLogEntry) {
	key, isKnown := f.position.Keys.Get(entry.GetEpoch())
	if !f.position.IsActive || !isKnown {
		return
	}
	if body, err := OpenBody(key, f.rules.ShareRootID, entry.GetEpoch(), entry.GetEncShareBody()); err == nil {
		f.position.Name = body.Name
	}
}

func (f *Follower) walkBack() error {
	for epoch := f.position.Epoch; epoch >= 1; epoch-- {
		newer, isNewer := f.position.Keys.Get(epoch)
		if _, isOlder := f.position.Keys.Get(epoch - 1); !isNewer || isOlder {
			continue
		}
		sealed, isSealed := f.previous[epoch]
		if !isSealed {
			continue
		}
		older, err := OpenPreviousKey(newer, f.rules.ShareRootID, epoch, sealed)
		if err != nil {
			return fmt.Errorf("%w: the key of epoch %d: %w", ErrDamagedLog, epoch-1, err)
		}
		f.position.Keys = f.position.Keys.Add(epoch-1, older)
	}
	return nil
}

// SealToReader seals the key of the epoch to the account box key of the reader, for the sync plugin.
func SealToReader(random io.Reader, box [32]byte, shareID, reader []byte, epoch uint32, key [32]byte) ([]byte, error) {
	plain := append(getBigEndian(epoch), key[:]...)
	return identity.SealToAccountBox(random, box, syncprotocol.SignInPluginID, GetSealContext(shareID, reader, epoch), plain)
}

// OpenSeal opens the seal of an epoch key addressed to the reader.
func OpenSeal(ctx context.Context, open Opener, shareID, reader []byte, epoch uint32, sealed []byte) ([32]byte, error) {
	plain, err := open(ctx, GetSealContext(shareID, reader, epoch), sealed)
	if err != nil {
		return [32]byte{}, err
	}
	if len(plain) != sealedKeySize || binary.BigEndian.Uint32(plain[:4]) != epoch {
		return [32]byte{}, ErrCannotOpen
	}
	return [32]byte(plain[4:]), nil
}

// GetMetaHash returns the SHA-256 of the sealed metadata of a revision.
func GetMetaHash(encMeta []byte) [32]byte { return sha256.Sum256(encMeta) }

var (
	errNoSeal        = errors.New("no seal for this reader")
	errWrongVersion  = errors.New("the version does not follow the one before")
	errWrongKind     = errors.New("the kind does not fit the place in the log")
	errWrongPrevious = errors.New("the previous hash is not the id of the entry before")
	errWrongShare    = errors.New("the entry belongs to another share")
	errEpochDown     = errors.New("the epoch goes down")
	errUnknownAuthor = errors.New("the author is not a device of the owner")
	errBadSignature  = errors.New("the signature is not valid")
)

// ListEntriesAfter returns the entries after the one a reader verified last. The entries start at that one: the server
// must serve it again, with the id the reader stored, or it has taken the log back to an older state or made another
// one (ErrRolledBack).
func ListEntriesAfter(entries []*syncv1.ShareLogEntry, version uint64, headID [32]byte) ([]*syncv1.ShareLogEntry, error) {
	if len(entries) == 0 || entries[0].GetVersion() != version || syncprotocol.GetShareLogEntryID(entries[0]) != headID {
		return nil, fmt.Errorf("%w: the entry %d the reader holds is not at the head of what the server serves", ErrRolledBack, version)
	}
	return entries[1:], nil
}
