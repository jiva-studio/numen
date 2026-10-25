// Package flashcardsstudio holds the manifest of the flashcards-studio plugin, for a host that links the plugin in.
package flashcardsstudio

import _ "embed"

// Manifest is the plugin's manifest.json.
//
//go:embed manifest.json
var Manifest []byte
