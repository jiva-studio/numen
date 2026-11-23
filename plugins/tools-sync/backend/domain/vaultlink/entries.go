package vaultlink

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

// Authors tells which signing keys belong to the devices of the account.
type Authors interface {
	IsTrusted(signKey []byte) bool
	// IsKnown reports whether the key is the signing key of a device that the chain ever listed, revoked or not.
	IsKnown(signKey []byte) bool
}

// ErrInvalidCatalog means the catalog of the server is not a signed, unbroken log of the devices of the account.
var ErrInvalidCatalog = errors.New("the catalog of the account is not valid")

var (
	errWrongVersion  = errors.New("the version does not follow the one before")
	errWrongPrevious = errors.New("the previous hash is not the id of the entry before")
	errBadSignature  = errors.New("the signature is not valid")
	errUnknownAuthor = errors.New("the author is not a device of the account")
)

// CheckEntries verifies that the entries are the unbroken log of the catalog, each signed by a device of the account. The log is hash-linked,
// so an entry of a trusted device vouches for every entry before it: those may be signed by a device that was revoked since, but the last
// entries, after the last one of a trusted device, may not.
func CheckEntries(entries []*syncv1.CatalogEntry, authors Authors) error {
	var previous [32]byte
	lastTrusted := -1
	for i, entry := range entries {
		if authors.IsTrusted(entry.GetAuthorDeviceSignKey()) {
			lastTrusted = i
		}
	}
	for i, entry := range entries {
		isVouched := i < lastTrusted
		if err := checkEntry(entry, uint64(i)+1, previous, authors, isVouched); err != nil {
			return fmt.Errorf("%w: entry %d: %w", ErrInvalidCatalog, i+1, err)
		}
		previous = syncprotocol.GetCatalogEntryID(entry)
	}
	return nil
}

func checkEntry(entry *syncv1.CatalogEntry, version uint64, previous [32]byte, authors Authors, isVouched bool) error {
	if err := syncprotocol.CheckCatalogEntry(entry); err != nil {
		return err
	}
	if entry.GetVersion() != version {
		return errWrongVersion
	}
	if version > 1 && !bytes.Equal(entry.GetPrevHash(), previous[:]) {
		return errWrongPrevious
	}
	if len(entry.GetSignature()) != 64 || !identity.VerifyForPlugin(entry.GetAuthorDeviceSignKey(), syncprotocol.SignInPluginID,
		syncprotocol.GetCatalogEntryMessage(entry), [64]byte(entry.GetSignature())) {
		return errBadSignature
	}
	author := entry.GetAuthorDeviceSignKey()
	if !authors.IsTrusted(author) && !(isVouched && authors.IsKnown(author)) {
		return errUnknownAuthor
	}
	return nil
}
