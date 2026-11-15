// Package reranking holds the manifest of the reranking plugin, for a host that links the plugin in.
package reranking

import _ "embed"

// Manifest is the plugin's manifest.json.
//
//go:embed manifest.json
var Manifest []byte
