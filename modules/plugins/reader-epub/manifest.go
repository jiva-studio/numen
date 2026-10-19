// Package readerepub holds the manifest of the EPUB reader plugin, for a host that links the plugin in.
package readerepub

import _ "embed"

// Manifest is the plugin's manifest.json.
//
//go:embed manifest.json
var Manifest []byte
