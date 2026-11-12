// Package protocol holds the capability names and wire contracts shared by hosts and plugins.
package protocol

import "strings"

// CapabilityAgentProvider is the capability of a plugin that runs agent programs.
const CapabilityAgentProvider = "ai.agent.provider"

// searchSourceReadyPrefix starts the event the host emits when a search source has indexed every chunk of a vault.
const searchSourceReadyPrefix = "onSearchSourceReady:"

// GetSearchSourceReadyEvent is the event the host emits to plugins when the search source has indexed every
// chunk of the vault of the given identity, but the chunks of a source whose indexing a person canceled,
// which stay out until the source changes. The host emits it after each indexing pass that leaves no other
// chunk pending, so it repeats; the state it reports is GetEmbeddingState of HostSearchService, which a
// plugin that starts later reads as well.
func GetSearchSourceReadyEvent(vaultID string) string {
	return searchSourceReadyPrefix + vaultID
}

// ReadSearchSourceReadyEvent returns the identity of the vault that a search-source-ready event names, and
// false for any other event.
func ReadSearchSourceReadyEvent(event string) (string, bool) {
	vaultID, found := strings.CutPrefix(event, searchSourceReadyPrefix)
	if !found || vaultID == "" {
		return "", false
	}
	return vaultID, true
}
