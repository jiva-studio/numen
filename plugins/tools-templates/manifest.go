// Package templates holds the manifest of the templates plugin, for a host that links the plugin in.
package templates

import _ "embed"

// Manifest is the plugin's manifest.json.
//
//go:embed manifest.json
var Manifest []byte
