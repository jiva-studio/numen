// Package connectionsllm holds the manifest of the plugin that provides the OpenRouter, OpenAI and
// OpenAI-compatible connection types, for a host that links the plugin in.
package connectionsllm

import _ "embed"

// Manifest is the plugin's manifest.json.
//
//go:embed manifest.json
var Manifest []byte
