package pluginhostsize

import (
	"go/ast"
	"go/types"
	"slices"
	"strconv"
	"strings"
	"testing"
	"golang.org/x/tools/go/packages"
	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// The roles of the plugin host are cut so that each does one thing; the limits are what the largest of
// them holds. A struct over them has taken a second job.
const (
	maxPluginAdapterMethods = 24
	maxPluginAdapterFields  = 16

	pluginAdapterPath = "internal/adapter/plugin"
)

// structSizeCount is what one walk over structs read.
type structSizeCount struct {
	structs     int
	mostMethods int
	mostFields  int
}

func isPluginAdapterPackage(pkg *packages.Package) bool {
	return strings.HasSuffix(pkg.PkgPath, "/"+pluginAdapterPath)
}

func getOversizedStructs(pkgs []*packages.Package, maxMethods, maxFields int) ([]load.Finding, structSizeCount) {
	var found []load.Finding
	var count structSizeCount
	for _, pkg := range pkgs {
		if !isPluginAdapterPackage(pkg) {
			continue
		}
		for _, file := range pkg.Syntax {
			if ast.IsGenerated(file) {
				continue
			}
			ast.Inspect(file, func(node ast.Node) bool {
				spec, ok := node.(*ast.TypeSpec)
				if !ok {
					return true
				}
				object, ok := pkg.TypesInfo.Defs[spec.Name].(*types.TypeName)
				if !ok || object.IsAlias() {
					return true
				}
				named, ok := object.Type().(*types.Named)
				if !ok {
					return true
				}
				shape, ok := named.Underlying().(*types.Struct)
				if !ok {
					return true
				}
				count.structs++
				methods, fields := named.NumMethods(), shape.NumFields()
				count.mostMethods = max(count.mostMethods, methods)
				count.mostFields = max(count.mostFields, fields)
				if methods > maxMethods || fields > maxFields {
					found = append(found, load.NewFinding(pkg.Fset, spec.Name.Pos(),
						pkg.Types.Name()+"."+object.Name()+" has "+strconv.Itoa(methods)+" methods and "+strconv.Itoa(fields)+" fields"))
				}
				return true
			})
		}
	}
	return load.SortFindings(found), count
}

func checkThePluginAdapterHasNoGodStruct(t *testing.T, workspace load.WorkspacePackages) {
	t.Helper()
	found, count := getOversizedStructs(load.GetModulePackages(t, workspace, "libs/core/kernel"), maxPluginAdapterMethods, maxPluginAdapterFields)
	for _, one := range found {
		t.Errorf("%s: %s, more than the %d methods and %d fields a struct of the plugin adapter may have",
			one.Position, one.Name, maxPluginAdapterMethods, maxPluginAdapterFields)
	}

	if count.structs == 0 {
		t.Fatalf("%d structs read, the most methods %d and the most fields %d: the rule is checked against nothing",
			count.structs, count.mostMethods, count.mostFields)
	}
}
