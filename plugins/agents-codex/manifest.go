// Package agentcodex holds the manifest of the OpenAI Codex agent plugin, for a host that links the plugin in.
package agentcodex

import _ "embed"

// Manifest is the plugin's manifest.json.
//
//go:embed manifest.json
var Manifest []byte
