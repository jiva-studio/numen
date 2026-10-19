// Package agentacp holds the manifest of the ACP agents plugin, for a host that links the plugin in.
package agentacp

import _ "embed"

// Manifest is the plugin's manifest.json.
//
//go:embed manifest.json
var Manifest []byte
