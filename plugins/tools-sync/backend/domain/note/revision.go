package note

import (
	"bytes"
	"errors"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

// ErrUnverified means a revision is not signed by a device of the account, is not what its id says, or does not open
// with the vault key.
var ErrUnverified = errors.New("the revision does not verify")

// OpenRevision checks the id and the signature of a stored revision and that isTrusted names its author, and opens the
// metadata of a live revision with the vault key. A removal has no metadata and gives an empty Meta.
func OpenRevision(vaultKey [32]byte, stored *syncv1.StoredRevision, isTrusted func(deviceSignKey []byte) bool) (Meta, error) {
	if err := CheckRevision(stored, isTrusted); err != nil {
		return Meta{}, err
	}
	rev := stored.GetRevision()
	if rev.GetDeleted() {
		return Meta{}, nil
	}
	meta, err := OpenMeta(vaultKey, rev.GetVaultId(), rev.GetFileId(), rev.GetEncMeta())
	if err != nil || len(meta.Key) != 32 {
		return Meta{}, ErrUnverified
	}
	if meta.Path, err = NormalizePath(meta.Path); err != nil {
		return Meta{}, ErrUnverified
	}
	return meta, nil
}

// CheckRevision checks the id and the signature of a stored revision and that isTrusted names its author.
func CheckRevision(stored *syncv1.StoredRevision, isTrusted func(deviceSignKey []byte) bool) error {
	rev := stored.GetRevision()
	id := syncprotocol.GetRevisionID(rev)
	if syncprotocol.CheckRevision(rev) != nil || !bytes.Equal(stored.GetRevId(), id[:]) || len(rev.GetSignature()) != 64 ||
		!identity.VerifyForPlugin(rev.GetAuthorDeviceSignKey(), syncprotocol.SignInPluginID, syncprotocol.GetRevisionMessage(rev), [64]byte(rev.GetSignature())) ||
		!isTrusted(rev.GetAuthorDeviceSignKey()) {
		return ErrUnverified
	}
	return nil
}
