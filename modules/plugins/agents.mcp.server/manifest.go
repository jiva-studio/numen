// Package agentmcpserver holds the manifest of the MCP server plugin.
package agentmcpserver

import _ "embed"

// Manifest is the plugin's manifest.json.
//
//go:embed manifest.json
var Manifest []byte
