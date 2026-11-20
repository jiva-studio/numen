// Package viewstodos holds the manifest of the todos plugin, for a host that links the plugin in.
package viewstodos

import _ "embed"

// Manifest is the plugin's manifest.json.
//
//go:embed manifest.json
var Manifest []byte
