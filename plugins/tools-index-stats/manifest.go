// Package indexstats holds the manifest of the index stats plugin, for a host that links the plugin in.
package indexstats

import _ "embed"

// Manifest is the plugin's manifest.json.
//
//go:embed manifest.json
var Manifest []byte
