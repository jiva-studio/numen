package interfacesize

import (
	"errors"
	"fmt"
	"go/ast"
	"go/types"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
	"golang.org/x/tools/go/packages"
	"gopkg.in/yaml.v3"
)

// structureConfigName is the golangci-lint configuration of the production-only rules, interfacebloat among them.
const structureConfigName = ".golangci-structure.yml"

func readInterfaceMax(path string) (int, bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, false, err
	}
	var config struct {
		Linters struct {
			Enable   []string `yaml:"enable"`
			Settings struct {
				Interfacebloat struct {
					Max *int `yaml:"max"`
				} `yaml:"interfacebloat"`
			} `yaml:"settings"`
		} `yaml:"linters"`
	}
	if err := yaml.Unmarshal(raw, &config); err != nil {
		return 0, false, err
	}
	if !slices.Contains(config.Linters.Enable, "interfacebloat") {
		return 0, false, nil
	}
	if config.Linters.Settings.Interfacebloat.Max == nil {
		return 0, false, fmt.Errorf("%s enables interfacebloat and sets no max", path)
	}
	return *config.Linters.Settings.Interfacebloat.Max, true, nil
}

func findStructureConfig(dir, root string) (string, error) {
	for current := dir; ; current = filepath.Dir(current) {
		path := filepath.Join(current, structureConfigName)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
		if current == root || filepath.Dir(current) == current {
			return "", nil
		}
	}
}

// interfaceCount is what one walk over interfaces read.
type interfaceCount struct {
	interfaces int
	composed   int
}

func getBloatedInterfaces(pkgs []*packages.Package, limit int) ([]load.Finding, interfaceCount) {
	var found []load.Finding
	var count interfaceCount
	for _, pkg := range pkgs {
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
				iface, ok := object.Type().Underlying().(*types.Interface)
				if !ok {
					return true
				}
				count.interfaces++
				if iface.NumEmbeddeds() > 0 {
					count.composed++
				}
				if iface.NumMethods() > limit {
					found = append(found, load.NewFinding(pkg.Fset, spec.Name.Pos(),
						pkg.Types.Name()+"."+object.Name()+" has "+strconv.Itoa(iface.NumMethods())+" methods"))
				}
				return true
			})
		}
	}
	return load.SortFindings(found), count
}

func isInterfaceSizeScope(rel string) bool {
	return slices.ContainsFunc([]string{"libs/core/kernel", "libs/note", "libs/plugin/sdk-go", "apps", "plugins"}, func(root string) bool {
		return load.IsWithin(filepath.ToSlash(rel), root)
	})
}

func checkInterfaceSizeCountsTheWholeMethodSet(t *testing.T, workspace load.WorkspacePackages) {
	t.Helper()
	root, err := filepath.Abs(load.RepoRootDir)
	if err != nil {
		t.Fatal(err)
	}
	var total interfaceCount
	covered := 0
	for _, dir := range load.ListWorkspaceModules(t) {
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			t.Fatal(err)
		}
		if !isInterfaceSizeScope(rel) {
			continue
		}
		config, err := findStructureConfig(dir, root)
		if err != nil {
			t.Fatal(err)
		}
		if config == "" {
			continue
		}
		limit, isEnabled, err := readInterfaceMax(config)
		if err != nil {
			t.Fatal(err)
		}
		if !isEnabled {
			continue
		}
		covered++
		found, count := getBloatedInterfaces(load.ListProductionPackages(workspace.ListModule(t, dir)), limit)
		total.interfaces += count.interfaces
		total.composed += count.composed
		for _, one := range found {
			t.Errorf("%s: %s, more than the %d %s allows", one.Position, one.Name, limit, config)
		}
	}

	// A walk that reads no interface built from embedded ones checks nothing the linter does not.
	if covered < 8 || total.interfaces < 100 || total.composed == 0 {
		t.Fatalf("%d modules, %d interfaces and %d composed interfaces read: the rule is checked against nothing",
			covered, total.interfaces, total.composed)
	}
}
