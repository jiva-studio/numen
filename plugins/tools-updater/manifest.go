// Package updater holds the manifest of the updater plugin, for a host that links the plugin in.
package updater

import _ "embed"

// Manifest is the plugin's manifest.json.
//
//go:embed manifest.json
var Manifest []byte
