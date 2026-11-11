package identities

import (
	"go/ast"
	"strings"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

func getNumberedTypes() map[string]bool {
	return map[string]bool{
		"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
		"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
		"uintptr": true, "byte": true, "rune": true,
	}
}

func getIdentities(file *ast.File) []load.Identity {
	return load.GetFieldsNamed(file, "ID")
}

func getMethodIdentities(file *ast.File) []load.Identity {
	var found []load.Identity
	ast.Inspect(file, func(node ast.Node) bool {
		spec, is := node.(*ast.TypeSpec)
		if !is {
			return true
		}
		target, is := spec.Type.(*ast.InterfaceType)
		if !is {
			return true
		}
		for _, method := range target.Methods.List {
			signature, is := method.Type.(*ast.FuncType)
			if !is || len(method.Names) == 0 {
				continue
			}
			fields := signature.Params.List
			if signature.Results != nil {
				fields = append(fields, signature.Results.List...)
			}
			for _, one := range fields {
				for _, name := range one.Names {
					if strings.HasSuffix(name.Name, "ID") {
						found = append(found, load.Identity{
							spec.Name.Name + "." + method.Names[0].Name,
							name.Name,
							load.GetTypeName(one.Type),
						})
					}
				}
			}
		}
		return true
	})
	return found
}
