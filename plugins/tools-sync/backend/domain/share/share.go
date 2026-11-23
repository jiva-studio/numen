// Package share is what Numen Sync knows of a shared folder: the keys of its epochs, what is sealed under them, the
// codes two people exchange and the rule that keeps shared folders apart.
package share

import (
	"errors"
	"io"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
)

var (
	// ErrWholeVault means the folder is the root of the vault.
	ErrWholeVault = errors.New("choose a folder, not the whole vault")
	// ErrInsideShare means the folder is a shared folder or lies inside one.
	ErrInsideShare = errors.New("a folder inside a shared folder cannot be shared")
	// ErrHoldsShare means the folder holds a shared folder.
	ErrHoldsShare = errors.New("a folder that holds a shared folder cannot be shared")
	// ErrCannotOpen means sealed share data does not open with the key, the share, the file and the epoch given.
	ErrCannotOpen = errors.New("the shared data cannot be opened")
	// ErrBadCode means a reader code or an invitation does not parse.
	ErrBadCode = errors.New("this is not a reader code or an invitation")
)

var errNotImplemented = errors.New("not implemented")

// Body is what a reader sees of a share before it has any file: the name of its vault.
type Body struct {
	Name string `json:"name"`
}

// Invitation is what the owner gives a reader: the share, the fingerprint of the reader it is for and the chain of
// the owner.
type Invitation struct {
	ShareRootID       []byte
	ReaderFingerprint identity.Fingerprint
	OwnerChain        []byte
}

// CheckNesting returns nil when folder may be shared next to the shared folders given: not the vault root, not a
// shared folder or inside one, not holding one. Folders are paths of the vault, compared by segment and without
// case.
func CheckNesting(_ string, _ []string) error { return errNotImplemented }

// GetRelativePath returns the path of a file relative to the shared folder, and false when the file is not under it.
func GetRelativePath(_, _ string) (string, bool) { return "", false }

// GetShareName returns the name a reader gives the vault of the folder: its last segment.
func GetShareName(_ string) string { return "" }

// SealMeta returns the metadata of a file under a share sealed with the key of the epoch, bound to the share, the
// file and the epoch. The path in meta is relative to the shared folder.
func SealMeta(_ io.Reader, _ [32]byte, _, _ []byte, _ uint32, _ note.Meta) ([]byte, error) {
	return nil, errNotImplemented
}

// OpenMeta opens what SealMeta sealed. Any other key, share, file or epoch is ErrCannotOpen.
func OpenMeta(_ [32]byte, _, _ []byte, _ uint32, _ []byte) (note.Meta, error) {
	return note.Meta{}, errNotImplemented
}

// SealBody returns the share body sealed with the key of the epoch.
func SealBody(_ io.Reader, _ [32]byte, _ []byte, _ uint32, _ Body) ([]byte, error) {
	return nil, errNotImplemented
}

// OpenBody opens what SealBody sealed. Any other key, share or epoch is ErrCannotOpen.
func OpenBody(_ [32]byte, _ []byte, _ uint32, _ []byte) (Body, error) {
	return Body{}, errNotImplemented
}

// SealPreviousKey seals the key of epoch-1 under the key of the epoch, as the prev_key_seal of an entry carries it.
func SealPreviousKey(_ io.Reader, _, _ [32]byte, _ []byte, _ uint32) ([]byte, error) {
	return nil, errNotImplemented
}

// OpenPreviousKey opens what SealPreviousKey sealed with the key of the epoch.
func OpenPreviousKey(_ [32]byte, _ []byte, _ uint32, _ []byte) ([32]byte, error) {
	return [32]byte{}, errNotImplemented
}

// GetSealContext returns the context the seal of an epoch key for a reader is made under.
func GetSealContext(_, _ []byte, _ uint32) []byte { return nil }

// GetOwnerBodyContext returns the context the owner body of an entry is sealed under.
func GetOwnerBodyContext(_ []byte, _ uint64) []byte { return nil }

// NewReaderCode returns the code a reader gives the owner: the chain of the reader.
func NewReaderCode(_ []byte) string { return "" }

// ParseReaderCode returns the chain a reader code holds, verified.
func ParseReaderCode(_ string) (identity.Chain, error) {
	return identity.Chain{}, errNotImplemented
}

// NewInvitation returns the text of an invitation.
func NewInvitation(_ Invitation) string { return "" }

// ParseInvitation reads the text of an invitation.
func ParseInvitation(_ string) (Invitation, error) { return Invitation{}, errNotImplemented }
