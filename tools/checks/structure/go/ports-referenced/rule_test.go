package portsreferenced

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

func getUncalledMethods(root, dir string) ([]string, error) {
	declared, err := getDeclaredMethods(dir)
	if err != nil {
		return nil, err
	}
	if len(declared) == 0 {
		return nil, fmt.Errorf("%s declares no port method: the walk is not reading it", dir)
	}

	called := map[string]bool{}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		if strings.HasSuffix(path, "_test.go") || load.IsUnder(filepath.Dir(path), filepath.Clean(dir)) {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(node ast.Node) bool {
			if pos, is := node.(*ast.SelectorExpr); is {
				called[pos.Sel.Name] = true
			}
			return true
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	var uncalled []string
	for one, method := range declared {
		if !called[method] {
			uncalled = append(uncalled, one)
		}
	}
	slices.Sort(uncalled)
	return uncalled, nil
}

func getDeclaredMethods(dir string) (map[string]string, error) {
	target, err := load.ReadTree(dir)
	if err != nil {
		return nil, err
	}
	declared := map[string]string{}
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
				shape, is := named.Type.(*ast.InterfaceType)
				if !is {
					continue
				}
				for _, method := range shape.Methods.List {
					if _, is := method.Type.(*ast.FuncType); !is {
						continue
					}
					for _, item := range method.Names {
						if item.IsExported() {
							declared[named.Name.Name+"."+item.Name] = item.Name
						}
					}
				}
			}
		}
	}
	return declared, nil
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
