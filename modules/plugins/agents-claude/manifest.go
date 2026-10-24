// Package agentclaude holds the manifest of the Claude Code agent plugin, for a host that links the plugin in.
package agentclaude

import _ "embed"

// Manifest is the plugin's manifest.json.
//
//go:embed manifest.json
var Manifest []byte
