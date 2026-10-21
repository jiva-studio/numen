// Package ocr holds the manifest of the OCR plugin, for a host that links the plugin in.
package ocr

import _ "embed"

// Manifest is the plugin's manifest.json.
//
//go:embed manifest.json
var Manifest []byte
