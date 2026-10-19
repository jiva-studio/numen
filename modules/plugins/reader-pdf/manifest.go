// Package readerpdf holds the manifest of the PDF reader plugin, for a host that links the plugin in.
package readerpdf

import _ "embed"

// Manifest is the plugin's manifest.json.
//
//go:embed manifest.json
var Manifest []byte
