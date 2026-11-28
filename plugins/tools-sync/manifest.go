// Package sync holds the manifest of the sync plugin, for a host that links the plugin in.
package sync

import _ "embed"

// Manifest is the plugin's manifest.json.
//
//go:embed manifest.json
var Manifest []byte
