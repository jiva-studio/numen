// Package taxonomy holds the manifest of the tags plugin, for a host that links the plugin in.
package taxonomy

import _ "embed"

// Manifest is the plugin's manifest.json.
//
//go:embed manifest.json
var Manifest []byte
