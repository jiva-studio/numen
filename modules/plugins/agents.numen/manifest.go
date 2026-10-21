// Package agent holds the manifest of the Numen Agent plugin, for a host that links the plugin in.
package agent

import _ "embed"

// Manifest is the plugin's manifest.json.
//
//go:embed manifest.json
var Manifest []byte
