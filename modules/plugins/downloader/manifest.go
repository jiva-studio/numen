// Package downloader holds the manifest of the media downloader plugin, for a host that links the plugin in.
package downloader

import _ "embed"

// Manifest is the plugin's manifest.json.
//
//go:embed manifest.json
var Manifest []byte
