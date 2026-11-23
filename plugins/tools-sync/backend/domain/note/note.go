// Package note is what Numen Sync knows of a note: its hash, its path as the server sees it, the metadata
// sealed with each revision and the rule that decides what a pull does with a note.
package note

import (
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

const (
	pathHashInfo = "numen-sync/path-hash/v1"
	extension    = ".md"
)

// ErrInvalidPath means a path is empty, absolute or leaves the vault.
var ErrInvalidPath = errors.New("the path is not inside the vault")

// Decision is what a pull does with a note whose head changed on the server.
type Decision int

const (
	// Adopt records the head as the base of the local note, which already holds it.
	Adopt Decision = iota + 1
	// Download replaces the local note, or makes it, with the head.
	Download
	// Conflict leaves the local note as it is.
	Conflict
)

// Hash returns the hexadecimal SHA-256 of the content.
func Hash(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// IsMarkdown reports whether the path names a markdown note.
func IsMarkdown(notePath string) bool { return strings.EqualFold(path.Ext(notePath), extension) }

// NormalizePath returns the path in Unicode normal form C, or ErrInvalidPath for a path that is not
// inside the vault.
func NormalizePath(notePath string) (string, error) {
	cleaned := path.Clean(norm.NFC.String(notePath))
	if notePath == "" || strings.HasPrefix(cleaned, "/") || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", ErrInvalidPath
	}
	return norm.NFC.String(notePath), nil
}

// HashPath returns the keyed hash of the path folded to one case in normal form C, so two paths that differ in
// case or composition only hash alike.
func HashPath(vaultKey [32]byte, notePath string) []byte {
	derived, err := hkdf.Key(sha256.New, vaultKey[:], nil, pathHashInfo, sha256.Size)
	if err != nil {
		return nil
	}
	mac := hmac.New(sha256.New, derived)
	mac.Write([]byte(cases.Fold().String(norm.NFC.String(notePath))))
	return mac.Sum(nil)
}

// Decide applies the rule of three hashes: local is the hash of the local note, empty when it is absent, base is the
// hash both computers last agreed on and head is the hash on the server. A head equal to the base is a revision this
// device pushed itself, so a local change after it is no conflict.
func Decide(local, base, head string) Decision {
	switch {
	case local == head, local != "" && head == base:
		return Adopt
	case local == "" || local == base:
		return Download
	}
	return Conflict
}
