// Package vaulttools holds the manifest of the vault tools plugin.
package vaulttools

import _ "embed"

// Manifest is the plugin's manifest.json.
//
//go:embed manifest.json
var Manifest []byte
