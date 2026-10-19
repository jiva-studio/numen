// Package proofreading holds the manifest of the proofreading plugin, for a host that links the plugin in.
package proofreading

import _ "embed"

// Manifest is the plugin's manifest.json.
//
//go:embed manifest.json
var Manifest []byte
