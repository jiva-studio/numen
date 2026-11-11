package container

import (
	"go/ast"
	"go/types"
	"slices"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
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

func getOversizedStructs(pkgs []*packages.Package, maxMethods, maxFields int) ([]finding, structSizeCount) {
	var found []finding
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
					found = append(found, newFinding(pkg.Fset, spec.Name.Pos(),
						pkg.Types.Name()+"."+object.Name()+" has "+strconv.Itoa(methods)+" methods and "+strconv.Itoa(fields)+" fields"))
				}
				return true
			})
		}
	}
	return sortFindings(found), count
}

func checkThePluginAdapterHasNoGodStruct(t *testing.T, workspace workspacePackages) {
	t.Helper()
	found, count := getOversizedStructs(getModulePackages(t, workspace, "libs/core/kernel"), maxPluginAdapterMethods, maxPluginAdapterFields)
	for _, one := range found {
		t.Errorf("%s: %s, more than the %d methods and %d fields a struct of the plugin adapter may have",
			one.position, one.name, maxPluginAdapterMethods, maxPluginAdapterFields)
	}

	if count.structs == 0 {
		t.Fatalf("%d structs read, the most methods %d and the most fields %d: the rule is checked against nothing",
			count.structs, count.mostMethods, count.mostFields)
	}
}

func TestPluginAdapterSizeRule_RejectsAStructOverTheLimits(t *testing.T) {
	dir := writeTreeFixture(t, map[string]string{
		"internal/adapter/plugin/host.go": `package plugin

type Many struct{ a, b, c int }

func (Many) One()   {}
func (Many) Two()   {}
func (Many) Three() {}

type Wide struct{ a, b, c, d int }

func (Wide) One() {}

type Small struct{ a int }

func (Small) One() {}
`,
		"internal/adapter/other/other.go": `package other

type Big struct{ a, b, c, d, e int }

func (Big) One() {}
`,
	})
	found, count := getOversizedStructs(loadFixture(t, dir), 2, 3)
	want := []string{
		"host.go:3 plugin.Many has 3 methods and 3 fields",
		"host.go:9 plugin.Wide has 1 methods and 4 fields",
	}
	if got := formatFindings(found); !slices.Equal(got, want) {
		t.Errorf("the rule rejects %v, want %v", got, want)
	}
	if count.structs != 3 || count.mostMethods != 3 || count.mostFields != 4 {
		t.Errorf("read %+v, want 3 structs of the plugin package, 3 methods and 4 fields at most", count)
	}
}
