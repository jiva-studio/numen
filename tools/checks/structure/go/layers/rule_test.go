package layers

import (
	"strings"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

func getLayers() map[string]bool {
	return map[string]bool{
		"adapter": true, "container": true, "domain": true, "port": true, "usecase": true,
	}
}

func getRefusal(from, to string) string {
	switch {
	case load.IsAdapter(from) || load.IsShared(from):
		if load.IsUnder(to, "container") {
			return "an adapter is given what it needs and assembles nothing"
		}
		if load.IsAdapter(to) && !isSibling(from, to) {
			return "an adapter is given what it needs and takes no other adapter"
		}
		if strings.HasPrefix(to, "usecase/") && !load.IsDriving(from) {
			return "a driven adapter is behind a port and runs no scenario"
		}
		if load.IsUnder(to, "port") && load.IsDriving(from) {
			return "a driving adapter calls the use case layer and imports no port directly"
		}
	case load.IsUnder(from, "container"):
		if !isAssembled(to) {
			return "the composition root assembles the core and does none of its work"
		}
	case load.IsUnder(from, "domain"):
		if !load.IsUnder(to, "domain") {
			return "a domain package imports the standard library, the notes domain and the domain packages, and nothing else of the core"
		}
	// Fixtures build the real adapters, and only a test is compiled from them.
	case strings.HasPrefix(from, "internal/test"):
	default:
		if load.IsAdapter(to) || load.IsUnder(to, "container") {
			return "the core reaches no adapter and nothing that assembles one"
		}
	}
	return ""
}

func isAssembled(to string) bool {
	return load.IsUnder(to, "container") || load.IsUnder(to, "domain") || load.IsUnder(to, "port") ||
		load.IsAdapter(to) || strings.HasPrefix(to, "usecase/")
}

func isSibling(from, to string) bool { return load.GetFamily(from) == load.GetFamily(to) }
