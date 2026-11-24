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
}

// ErrInvalidCatalog means the catalog of the server is not a signed, unbroken log of the devices of the account.
var ErrInvalidCatalog = errors.New("the catalog of the account is not valid")

var (
	errWrongVersion  = errors.New("the version does not follow the one before")
	errWrongPrevious = errors.New("the previous hash is not the id of the entry before")
	errBadSignature  = errors.New("the signature is not valid")
	errUnknownAuthor = errors.New("the author is not a device of the account")
)

// CheckEntries verifies that the entries are the unbroken log of the catalog, each signed by a device of the account.
func CheckEntries(entries []*syncv1.CatalogEntry, authors Authors) error {
	var previous [32]byte
	for i, entry := range entries {
		if err := checkEntry(entry, uint64(i)+1, previous, authors); err != nil {
			return fmt.Errorf("%w: entry %d: %w", ErrInvalidCatalog, i+1, err)
		}
		previous = syncprotocol.GetCatalogEntryID(entry)
	}
	return nil
}

func checkEntry(entry *syncv1.CatalogEntry, version uint64, previous [32]byte, authors Authors) error {
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
	if !authors.IsTrusted(entry.GetAuthorDeviceSignKey()) {
		return errUnknownAuthor
	}
	return nil
}
