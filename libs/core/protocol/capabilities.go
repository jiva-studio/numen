// Package protocol holds the wire contracts shared by hosts and plugins.
package protocol

import "strings"

// vectorsReadyPrefix starts the event the host emits when every chunk of a vault carries a vector.
const vectorsReadyPrefix = "onVectorsReady:"

// GetVectorsReadyEvent is the event the host emits to plugins when every chunk of the vault of the
// given identity carries a vector under the model in use, but the chunks of a source whose indexing a
// person canceled, which stay without one until the source changes. The host emits it after each
// indexing pass that leaves no other vector pending, so it repeats; the state it reports is GetEmbeddingState of
// HostSearchService, which a plugin that starts later reads as well.
func GetVectorsReadyEvent(vaultID string) string {
	return vectorsReadyPrefix + vaultID
}

// ReadVectorsReadyEvent returns the identity of the vault that a vectors-ready event names, and
// false for any other event.
func ReadVectorsReadyEvent(event string) (string, bool) {
	vaultID, found := strings.CutPrefix(event, vectorsReadyPrefix)
	if !found || vaultID == "" {
		return "", false
	}
	return vaultID, true
}
