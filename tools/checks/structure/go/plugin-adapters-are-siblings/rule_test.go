package pluginadaptersaresiblings

import (
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

const adapterSegment = "/backend/internal/adapter/"

func getAdapterName(pluginDir, path string) string {
	_, rest, found := strings.Cut(filepath.ToSlash(path), "plugins/"+pluginDir+adapterSegment)
	if !found {
		return ""
	}
	name, _, _ := strings.Cut(rest, "/")
	return name
}

func getSiblingAdapterImport(pluginDir, from, dep string) string {
	prefix := load.PluginsPrefix + pluginDir + adapterSegment
	rest, found := strings.CutPrefix(dep, prefix)
	if !found {
		return ""
	}
	name, _, _ := strings.Cut(rest, "/")
	if name == from {
		return ""
	}
	return name
}
