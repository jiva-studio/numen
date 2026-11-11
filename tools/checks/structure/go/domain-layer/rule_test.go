package domainlayer

import (
	"strings"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// notesDomain is the package of the note model that a domain package of the core may import.
const notesDomain = load.NotesModule + "/domain"

// catalogManifest is the library of the plugin contract that a domain package of the core may import; it
// never imports the core.
const catalogManifest = load.NumenRoot + "libs/plugin/catalog/manifest"

func isStandardLibrary(path string) bool {
	first, _, _ := strings.Cut(path, "/")
	return !strings.Contains(first, ".")
}

func getDomainImportRefusal(to string) string {
	switch {
	case isStandardLibrary(to), to == notesDomain, to == catalogManifest, strings.HasPrefix(to, load.Module+"domain/"):
		return ""
	case strings.HasPrefix(to, load.Module):
		return "a domain package names no layer of the core other than the domain"
	}
	return "a domain package imports the standard library, the notes domain, the manifest of the plugin catalog and the domain packages, and no third-party code"
}

func getCoreImportRefusal(to string) string {
	if strings.HasPrefix(to, load.Module) || to == strings.TrimSuffix(load.Module, "/") {
		return "the plugin SDK and the plugin catalog import no package of the core, a domain package included"
	}
	return ""
}
