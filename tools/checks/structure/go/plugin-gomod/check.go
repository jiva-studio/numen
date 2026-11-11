package plugingomod

import (
	"os"
	"path/filepath"
	"testing"
	"golang.org/x/mod/modfile"
	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

func getPluginGoModViolations(dir string, file *modfile.File) []string {
	var refusals []string
	mod := load.GetModulePath(file)
	if want := load.PluginsPrefix + dir; mod != want {
		refusals = append(refusals, "declares module "+mod+", want "+want)
	}
	for _, required := range file.Require {
		if required.Indirect {
			continue
		}
		if why := load.GetPluginDependencyRefusal(mod, required.Mod.Path); why != "" {
			refusals = append(refusals, "requires "+required.Mod.Path+": "+why)
		}
	}
	for _, replaced := range file.Replace {
		if replaced.New.Version == "" {
			refusals = append(refusals, "builds "+replaced.Old.Path+" from the folder "+replaced.New.Path+
				": a plugin builds in a repository of its own")
		}
	}
	return refusals
}
