// Package share is what Numen Sync knows of a shared folder: the keys of its epochs, what is sealed under them, the
// codes two people exchange and the rule that keeps shared folders apart.
package share

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/chacha20poly1305"

	"github.com/jiva-studio/numen/libs/shared/identity"
)

const (
	sealVersion      = 0x01
	metaLabel        = "numen-sync/share-meta/v1"
	bodyLabel        = "numen-sync/share-body/v1"
	previousKeyLabel = "numen-sync/share-prevkey/v1"
	sealLabel        = "numen-sync/share-seal/v1"
	ownerBodyLabel   = "numen-sync/share-owner/v1"
	readerPrefix     = "numen-reader-1."
	invitationPrefix = "numen-share-1."
	idSize           = 16
	fingerprintSize  = 16
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
	// errNoPreviousEpoch means the first epoch has no epoch before it.
	errNoPreviousEpoch = errors.New("the first epoch has no previous key")
)

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

// NormalizeFolder returns the folder without the slashes around it; the vault root is the empty string.
func NormalizeFolder(folder string) string {
	trimmed := strings.Trim(strings.ReplaceAll(folder, "\\", "/"), "/")
	if trimmed == "." {
		return ""
	}
	return trimmed
}

func getSegments(folder string) []string {
	normalized := NormalizeFolder(folder)
	if normalized == "" {
		return nil
	}
	return strings.Split(normalized, "/")
}

func hasPrefix(segments, prefix []string) bool {
	if len(prefix) > len(segments) {
		return false
	}
	for i, one := range prefix {
		if !strings.EqualFold(segments[i], one) {
			return false
		}
	}
	return true
}

// CheckNesting returns nil when folder may be shared next to the shared folders given: not the vault root, not a
// shared folder or inside one, not holding one. Folders are paths of the vault, compared by segment and without
// case.
func CheckNesting(folder string, shared []string) error {
	segments := getSegments(folder)
	if len(segments) == 0 {
		return ErrWholeVault
	}
	for _, one := range shared {
		other := getSegments(one)
		if len(other) == 0 {
			continue
		}
		if hasPrefix(segments, other) {
			return ErrInsideShare
		}
		if hasPrefix(other, segments) {
			return ErrHoldsShare
		}
	}
	return nil
}

func GetRelativePath(folder, filePath string) (string, bool) {
	prefix := getSegments(folder)
	segments := strings.Split(filePath, "/")
	if len(prefix) == 0 || len(segments) <= len(prefix) || !hasPrefix(segments, prefix) {
		return "", false
	}
	return strings.Join(segments[len(prefix):], "/"), true
}

// GetShareName returns the name a reader gives the vault of the folder: its last segment.
func GetShareName(folder string) string {
	segments := getSegments(folder)
	if len(segments) == 0 {
		return ""
	}
	return segments[len(segments)-1]
}

// CleanName returns the name a received vault gets on this computer: a name from an owner that holds no separator
// and is not empty.
func CleanName(name string) string {
	cleaned := strings.TrimSpace(strings.NewReplacer("/", "_", "\\", "_", ":", "_").Replace(name))
	if cleaned == "" || cleaned == "." || cleaned == ".." {
		return "Shared"
	}
	return cleaned
}

func getBigEndian(value uint32) []byte { return binary.BigEndian.AppendUint32(nil, value) }

func seal(random io.Reader, key [32]byte, associated, plain []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(key[:])
	if err != nil {
		return nil, fmt.Errorf("make the share cipher: %w", err)
	}
	nonce := make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := io.ReadFull(random, nonce); err != nil {
		return nil, fmt.Errorf("read a nonce: %w", err)
	}
	sealed := append([]byte{sealVersion}, nonce...)
	return aead.Seal(sealed, nonce, plain, associated), nil
}

func open(key [32]byte, associated, sealed []byte) ([]byte, error) {
	if len(sealed) < 1+chacha20poly1305.NonceSizeX || sealed[0] != sealVersion {
		return nil, ErrCannotOpen
	}
	aead, err := chacha20poly1305.NewX(key[:])
	if err != nil {
		return nil, fmt.Errorf("make the share cipher: %w", err)
	}
	nonce, body := sealed[1:1+chacha20poly1305.NonceSizeX], sealed[1+chacha20poly1305.NonceSizeX:]
	plain, err := aead.Open(nil, nonce, body, associated)
	if err != nil {
		return nil, ErrCannotOpen
	}
	return plain, nil
}

func getMetaData(shareID, fileID []byte, epoch uint32) []byte {
	return bytes.Join([][]byte{[]byte(metaLabel), shareID, fileID, getBigEndian(epoch)}, nil)
}

// SealMeta returns the metadata of a file under a share sealed with the key of the epoch, bound to the share, the
// file and the epoch. The path in meta is relative to the shared folder.
func SealMeta[M any](random io.Reader, key [32]byte, shareID, fileID []byte, epoch uint32, meta M) ([]byte, error) {
	plain, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("encode the metadata: %w", err)
	}
	return seal(random, key, getMetaData(shareID, fileID, epoch), plain)
}

// OpenMeta opens what SealMeta sealed. Any other key, share, file or epoch is ErrCannotOpen.
func OpenMeta[M any](key [32]byte, shareID, fileID []byte, epoch uint32, sealed []byte) (M, error) {
	var zero M
	plain, err := open(key, getMetaData(shareID, fileID, epoch), sealed)
	if err != nil {
		return zero, err
	}
	var meta M
	if err := json.Unmarshal(plain, &meta); err != nil {
		return zero, ErrCannotOpen
	}
	return meta, nil
}

func getBodyData(shareID []byte, epoch uint32) []byte {
	return bytes.Join([][]byte{[]byte(bodyLabel), shareID, getBigEndian(epoch)}, nil)
}

func SealBody(random io.Reader, key [32]byte, shareID []byte, epoch uint32, body Body) ([]byte, error) {
	plain, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode the share body: %w", err)
	}
	return seal(random, key, getBodyData(shareID, epoch), plain)
}

// OpenBody opens what SealBody sealed. Any other key, share or epoch is ErrCannotOpen.
func OpenBody(key [32]byte, shareID []byte, epoch uint32, sealed []byte) (Body, error) {
	plain, err := open(key, getBodyData(shareID, epoch), sealed)
	if err != nil {
		return Body{Name: ""}, err
	}
	var body Body
	if err := json.Unmarshal(plain, &body); err != nil {
		return Body{Name: ""}, ErrCannotOpen
	}
	return body, nil
}

func getPreviousKeyData(shareID []byte, epoch uint32) []byte {
	return bytes.Join([][]byte{[]byte(previousKeyLabel), shareID, getBigEndian(epoch - 1)}, nil)
}

// SealPreviousKey seals the key of epoch-1 under the key of the epoch, as the prev_key_seal of an entry carries it.
func SealPreviousKey(random io.Reader, newer, older [32]byte, shareID []byte, epoch uint32) ([]byte, error) {
	if epoch == 0 {
		return nil, errNoPreviousEpoch
	}
	return seal(random, newer, getPreviousKeyData(shareID, epoch), older[:])
}

func OpenPreviousKey(newer [32]byte, shareID []byte, epoch uint32, sealed []byte) ([32]byte, error) {
	if epoch == 0 {
		return [32]byte{}, errNoPreviousEpoch
	}
	plain, err := open(newer, getPreviousKeyData(shareID, epoch), sealed)
	if err != nil {
		return [32]byte{}, err
	}
	if len(plain) != len(newer) {
		return [32]byte{}, ErrCannotOpen
	}
	return [32]byte(plain), nil
}

// GetSealContext returns the context the seal of an epoch key for a reader is made under.
func GetSealContext(shareID, reader []byte, epoch uint32) []byte {
	return bytes.Join([][]byte{[]byte(sealLabel), shareID, reader, getBigEndian(epoch)}, nil)
}

// GetOwnerBodyContext returns the context the owner body of an entry is sealed under.
func GetOwnerBodyContext(shareID []byte, version uint64) []byte {
	return binary.BigEndian.AppendUint64(bytes.Join([][]byte{[]byte(ownerBodyLabel), shareID}, nil), version)
}

// NewReaderCode returns the code a reader gives the owner: the chain of the reader.
func NewReaderCode(chain []byte) string {
	return readerPrefix + base64.RawURLEncoding.EncodeToString(chain)
}

// ParseReaderCode returns the chain a reader code holds, verified.
func ParseReaderCode(text string) (identity.Chain, error) {
	encoded, ok := strings.CutPrefix(strings.TrimSpace(text), readerPrefix)
	if !ok {
		return identity.Chain{}, ErrBadCode
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return identity.Chain{}, fmt.Errorf("%w: %w", ErrBadCode, err)
	}
	chain, err := identity.ParseChain(raw)
	if err != nil {
		return identity.Chain{}, fmt.Errorf("%w: %w", ErrBadCode, err)
	}
	return chain, nil
}

// NewInvitation returns the text of an invitation.
func NewInvitation(invitation Invitation) string {
	raw := bytes.Join([][]byte{invitation.ShareRootID, invitation.ReaderFingerprint[:], invitation.OwnerChain}, nil)
	return invitationPrefix + base64.RawURLEncoding.EncodeToString(raw)
}

func ParseInvitation(text string) (Invitation, error) {
	encoded, ok := strings.CutPrefix(strings.TrimSpace(text), invitationPrefix)
	if !ok {
		return Invitation{}, ErrBadCode
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return Invitation{}, fmt.Errorf("%w: %w", ErrBadCode, err)
	}
	if len(raw) <= idSize+fingerprintSize {
		return Invitation{}, ErrBadCode
	}
	return Invitation{
		ShareRootID:       raw[:idSize],
		ReaderFingerprint: identity.Fingerprint(raw[idSize : idSize+fingerprintSize]),
		OwnerChain:        raw[idSize+fingerprintSize:],
	}, nil
}
