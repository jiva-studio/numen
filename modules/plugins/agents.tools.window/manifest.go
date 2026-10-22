// Package window holds the manifest of the Window Interaction Tools plugin.
package window

import _ "embed"

// Manifest is the plugin's manifest.json.
//
//go:embed manifest.json
var Manifest []byte
