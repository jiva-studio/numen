package manifest

// FileName is the name of the manifest file in the folder of a plugin.
const FileName = "manifest.json"

// Discovered is a validated plugin located on disk.
type Discovered struct {
	Dir      string
	Manifest Manifest
}

// FilterByTarget returns only plugins that can run in the target workspace.
func FilterByTarget(plugins []Discovered, target string) []Discovered {
	var matched []Discovered
	for _, p := range plugins {
		if p.Manifest.Supports(target) {
			matched = append(matched, p)
		}
	}
	return matched
}
