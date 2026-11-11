package pluginisolationpackages

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"golang.org/x/tools/go/packages"
	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

func getPluginCapture(pkgPath string) string {
	rest, load.IsPlugin := strings.CutPrefix(pkgPath, load.PluginsPrefix)
	if !load.IsPlugin {
		return ""
	}
	dir, _, _ := strings.Cut(rest, "/")
	dir = strings.TrimSuffix(strings.TrimSuffix(dir, ".test"), "_test")
	return dir
}

func isForeignToPlugins(own, pkgPath string) bool {
	if one := getPluginCapture(pkgPath); one != "" {
		return one != own
	}
	return pkgPath == strings.TrimSuffix(load.Module, "/") || strings.HasPrefix(pkgPath, load.Module) || strings.HasPrefix(pkgPath, load.NumenRoot+"apps/")
}

func getForeignDependencies(root *packages.Package) []string {
	own := getPluginCapture(root.PkgPath)
	visited := map[string]bool{}
	var foreign []string
	var walk func(pkg *packages.Package, isDirect bool)
	walk = func(pkg *packages.Package, isDirect bool) {
		if visited[pkg.ID] {
			return
		}
		visited[pkg.ID] = true
		if isForeignToPlugins(own, pkg.PkgPath) || (isDirect && getPluginCapture(pkg.PkgPath) != own && load.GetPluginDependencyRefusal(load.PluginsPrefix+own, pkg.PkgPath) != "") {
			foreign = append(foreign, pkg.PkgPath)
		}
		for _, imported := range pkg.Imports {
			walk(imported, false)
		}
	}
	for _, imported := range root.Imports {
		walk(imported, true)
	}
	slices.Sort(foreign)
	return slices.Compact(foreign)
}
