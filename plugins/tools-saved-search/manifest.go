// Package savedsearch holds the manifest of the saved search plugin.
package savedsearch

import _ "embed"

// Manifest is the plugin's manifest.json.
//
//go:embed manifest.json
var Manifest []byte
