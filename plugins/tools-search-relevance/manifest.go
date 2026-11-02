// Package searchrelevance holds the manifest of the search relevance plugin, for a host that links the plugin in.
package searchrelevance

import _ "embed"

// Manifest is the plugin's manifest.json.
//
//go:embed manifest.json
var Manifest []byte
