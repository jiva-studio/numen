// Package transcription holds the manifest of the transcription plugin, for a host that links the plugin in.
package transcription

import _ "embed"

// Manifest is the plugin's manifest.json.
//
//go:embed manifest.json
var Manifest []byte
