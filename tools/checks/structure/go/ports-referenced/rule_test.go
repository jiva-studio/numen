package portsreferenced

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
	"golang.org/x/tools/go/packages"
)

// getUncalledMethods loads the packages under root with type information for every shipped platform, since a
// call can sit in a file only one platform builds, and returns the methods of the port interfaces under dir that
// no package outside dir calls through the port.
func getUncalledMethods(root, dir string, env ...string) ([]string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	declared := map[string]bool{}
	called := map[string]bool{}
	for _, platform := range load.ListShippedPlatforms() {
		config := load.NewTypedConfig(root, slices.Concat(env, []string{"GOOS=" + platform.GOOS, "GOARCH=" + platform.GOARCH}))
		config.Tests = false
		pkgs, err := packages.Load(config, "./...")
		if err != nil {
			return nil, err
		}
		for _, pkg := range pkgs {
			if len(pkg.Errors) > 0 {
				return nil, fmt.Errorf("%s does not load for %s/%s: %v", pkg.ID, platform.GOOS, platform.GOARCH, pkg.Errors[0])
			}
		}
		methods := getPortMethods(pkgs, dir)
		for _, name := range methods {
			declared[name] = true
		}
		for _, pkg := range pkgs {
			if isUnderDir(pkg, dir) {
				continue
			}
			for _, file := range pkg.Syntax {
				ast.Inspect(file, func(node ast.Node) bool {
					if expr, is := node.(*ast.SelectorExpr); is {
						if name, is := getCalledPortMethod(pkg.TypesInfo, expr, methods); is {
							called[name] = true
						}
					}
					return true
				})
			}
		}
	}
	if len(declared) == 0 {
		return nil, fmt.Errorf("%s declares no port method: the walk is not reading it", dir)
	}

	var uncalled []string
	for one := range declared {
		if !called[one] {
			uncalled = append(uncalled, one)
		}
	}
	slices.Sort(uncalled)
	return uncalled, nil
}

// getPortMethods returns the exported methods that the exported interfaces under dir declare, named
// Interface.Method. A method an interface takes from an embedded interface belongs to the embedded one.
func getPortMethods(pkgs []*packages.Package, dir string) map[*types.Func]string {
	methods := map[*types.Func]string{}
	for _, pkg := range pkgs {
		if !isUnderDir(pkg, dir) {
			continue
		}
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			object, is := scope.Lookup(name).(*types.TypeName)
			if !is || !object.Exported() || object.IsAlias() {
				continue
			}
			shape, is := object.Type().Underlying().(*types.Interface)
			if !is {
				continue
			}
			for i := range shape.NumExplicitMethods() {
				if method := shape.ExplicitMethod(i); method.Exported() {
					methods[method] = name + "." + method.Name()
				}
			}
		}
	}
	return methods
}

// getCalledPortMethod returns the port method that expr selects. The selection resolves to the method object of
// the port when the receiver is the port, an interface that embeds it, or a type parameter it constrains. A
// method of the same name on a concrete type resolves to another object and does not count.
func getCalledPortMethod(info *types.Info, expr *ast.SelectorExpr, methods map[*types.Func]string) (string, bool) {
	selection, is := info.Selections[expr]
	if !is || selection.Kind() == types.FieldVal {
		return "", false
	}
	method, is := selection.Obj().(*types.Func)
	if !is {
		return "", false
	}
	name, is := methods[method.Origin()]
	return name, is
}

func isUnderDir(pkg *packages.Package, dir string) bool {
	return len(pkg.GoFiles) > 0 && load.IsUnder(filepath.ToSlash(filepath.Dir(pkg.GoFiles[0])), filepath.ToSlash(dir))
}

func getUnnamedTypes(root, dir string, onlyPorts bool) ([]string, error) {
	declared, err := getDeclaredTypes(dir)
	if err != nil {
		return nil, err
	}
	if len(declared) == 0 {
		return nil, fmt.Errorf("%s declares no type: the walk is not reading it", dir)
	}

	named := map[string]bool{}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		if load.IsUnder(load.GetPackage(root, path), "port") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(node ast.Node) bool {
			pos, is := node.(*ast.SelectorExpr)
			if !is {
				return true
			}
			if from, is := pos.X.(*ast.Ident); is && load.IsPortName(from.Name) {
				named[pos.Sel.Name] = true
			}
			return true
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	var unnamed []string
	for one, isPort := range declared {
		if onlyPorts && !isPort {
			continue
		}
		if !named[one] {
			unnamed = append(unnamed, one)
		}
	}
	slices.Sort(unnamed)
	return unnamed, nil
}

func getDeclaredTypes(dir string) (map[string]bool, error) {
	target, err := load.ReadTree(dir)
	if err != nil {
		return nil, err
	}
	declared := map[string]bool{}
	for _, one := range target {
		if one.IsDir() || !strings.HasSuffix(one.Name(), ".go") ||
			strings.HasSuffix(one.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, one.Name()), nil, 0)
		if err != nil {
			return nil, err
		}
		for _, decl := range file.Decls {
			pos, is := decl.(*ast.GenDecl)
			if !is || pos.Tok != token.TYPE {
				continue
			}
			for _, spec := range pos.Specs {
				named, is := spec.(*ast.TypeSpec)
				if !is || !named.Name.IsExported() {
					continue
				}
				_, isPort := named.Type.(*ast.InterfaceType)
				declared[named.Name.Name] = isPort
			}
		}
	}
	return declared, nil
}
