// Package protocol holds the wire contracts shared by hosts and plugins.
package protocol

import "strings"

// indexChangedPrefix starts the event the host emits when indexing changed the index of a vault.
const indexChangedPrefix = "onIndexChanged:"

// GetIndexChangedEvent is the event the host emits to plugins when an indexing pass changed the index of the
// vault of the given identity (notes, chunks or vectors saved or removed) and left nothing pending. A pass
// that changes nothing emits nothing. The event carries only the vault; a plugin reads the state through
// HostSearchService, GetEmbeddingState for the vectors.
func GetIndexChangedEvent(vaultID string) string {
	return indexChangedPrefix + vaultID
}

// ReadIndexChangedEvent returns the identity of the vault that an index-changed event names, and false for
// any other event.
func ReadIndexChangedEvent(event string) (string, bool) {
	vaultID, found := strings.CutPrefix(event, indexChangedPrefix)
	if !found || vaultID == "" {
		return "", false
	}
	return vaultID, true
}

// filesChangedPrefix starts the event the host emits when the index wrote changes into the file change feed of a vault.
const filesChangedPrefix = "onFilesChanged:"

// GetFilesChangedEvent is the event the host emits to plugins after a write of the index stored changes in the
// file change feed of the vault of the given identity. The host emits it once the write is committed, so the
// changes are readable through ListFileChanges; several writes close together are one event.
func GetFilesChangedEvent(vaultID string) string {
	return filesChangedPrefix + vaultID
}

// ReadFilesChangedEvent returns the identity of the vault that a files-changed event names, and false for any
// other event.
func ReadFilesChangedEvent(event string) (string, bool) {
	vaultID, found := strings.CutPrefix(event, filesChangedPrefix)
	if !found || vaultID == "" {
		return "", false
	}
	return vaultID, true
}
