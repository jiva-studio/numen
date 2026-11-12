// Package agentantigravity holds the manifest of the Antigravity agent plugin, for a host that links the plugin in.
package agentantigravity

import _ "embed"

// Manifest is the plugin's manifest.json.
//
//go:embed manifest.json
var Manifest []byte
