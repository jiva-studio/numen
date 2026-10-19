// Package embeddings holds the manifest of the embeddings plugin, for a host that links the plugin in.
package embeddings

import _ "embed"

// Manifest is the plugin's manifest.json.
//
//go:embed manifest.json
var Manifest []byte
