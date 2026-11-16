package load

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

// PortInterface is an interface that the core requires: a port.
type PortInterface struct {
	Name  string
	Iface *types.Interface
}

func IsInLayer(path, root, layer string) bool {
	return IsWithin(path, root+"/"+layer)
}

// ListDeclaredPortInterfaces lists the interfaces with methods that a package under port/ declares.
func ListDeclaredPortInterfaces(pkgs []*packages.Package, root string) []PortInterface {
	var found []PortInterface
	for _, pkg := range pkgs {
		if !IsInLayer(pkg.PkgPath, root, "port") {
			continue
		}
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			object, ok := scope.Lookup(name).(*types.TypeName)
			if !ok || object.IsAlias() {
				continue
			}
			iface, ok := object.Type().Underlying().(*types.Interface)
			if !ok || !iface.IsMethodSet() || iface.NumMethods() == 0 {
				continue
			}
			found = append(found, PortInterface{Name: pkg.PkgPath + "." + name, Iface: iface})
		}
	}
	return found
}

func GetImplementation(pkg *packages.Package, ports []PortInterface) Finding {
	scope := pkg.Types.Scope()
	for _, name := range scope.Names() {
		object, ok := scope.Lookup(name).(*types.TypeName)
		if !ok || object.IsAlias() || types.IsInterface(object.Type()) {
			continue
		}
		if named, ok := object.Type().(*types.Named); ok && named.TypeParams().Len() > 0 {
			continue
		}
		for _, port := range ports {
			if types.Implements(object.Type(), port.Iface) || types.Implements(types.NewPointer(object.Type()), port.Iface) {
				return NewFinding(pkg.Fset, object.Pos(), pkg.Types.Name()+"."+name+" implements "+port.Name)
			}
		}
	}
	return Finding{Position: "", Name: ""}
}

func getPortInterface(t types.Type, root string) (name string, iface *types.Interface) {
	named, ok := types.Unalias(t).(*types.Named)
	if !ok || named.Obj().Pkg() == nil {
		return "", nil
	}
	path := named.Obj().Pkg().Path()
	if !IsInLayer(path, root, "port") && !IsInLayer(path, root, "usecase") {
		return "", nil
	}
	iface, ok = named.Underlying().(*types.Interface)
	if !ok || !iface.IsMethodSet() || iface.NumMethods() == 0 {
		return "", nil
	}
	return path + "." + named.Obj().Name(), iface
}

func stripContainers(t types.Type) types.Type {
	for {
		switch next := types.Unalias(t).(type) {
		case *types.Pointer:
			t = next.Elem()
		case *types.Slice:
			t = next.Elem()
		case *types.Array:
			t = next.Elem()
		case *types.Map:
			t = next.Elem()
		case *types.Chan:
			t = next.Elem()
		default:
			return t
		}
	}
}

func isUseCaseTestSupport(pkg *packages.Package) bool {
	return strings.HasSuffix(pkg.Types.Name(), "test")
}

func listUseCaseHeldTypes(pkg *packages.Package) []types.Type {
	var found []types.Type
	for _, file := range pkg.Syntax {
		if strings.HasSuffix(pkg.Fset.Position(file.Pos()).Filename, "_test.go") {
			continue
		}
		for _, decl := range file.Decls {
			switch decl := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range decl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					if structType, ok := typeSpec.Type.(*ast.StructType); ok {
						for _, field := range structType.Fields.List {
							found = append(found, pkg.TypesInfo.TypeOf(field.Type))
						}
					}
				}
			case *ast.FuncDecl:
				if decl.Recv != nil || !decl.Name.IsExported() {
					continue
				}
				for _, param := range decl.Type.Params.List {
					found = append(found, pkg.TypesInfo.TypeOf(param.Type))
				}
			}
		}
	}
	return found
}

// ListPortInterfaces lists the ports of the core: the interfaces with methods under port/, and the named
// interfaces declared under port/ or usecase/ that a use case package holds as the type of a struct field or
// takes as a parameter of an exported function. The contract of a use case itself is held by no other
// use case, so it is not a port.
func ListPortInterfaces(pkgs []*packages.Package, root string) []PortInterface {
	found := ListDeclaredPortInterfaces(pkgs, root)
	seen := map[string]bool{}
	for _, one := range found {
		seen[one.Name] = true
	}
	for _, pkg := range pkgs {
		if !IsInLayer(pkg.PkgPath, root, "usecase") || isUseCaseTestSupport(pkg) {
			continue
		}
		for _, t := range listUseCaseHeldTypes(pkg) {
			if t == nil {
				continue
			}
			name, iface := getPortInterface(stripContainers(t), root)
			if iface != nil && !seen[name] {
				seen[name] = true
				found = append(found, PortInterface{Name: name, Iface: iface})
			}
		}
	}
	return found
}

// IsPortType reports whether the object is the type name of one of the ports.
func IsPortType(object types.Object, ports []PortInterface) bool {
	name, ok := object.(*types.TypeName)
	if !ok || name.Pkg() == nil {
		return false
	}
	full := name.Pkg().Path() + "." + name.Name()
	for _, port := range ports {
		if port.Name == full {
			return true
		}
	}
	return false
}

// ListReferences lists the identifiers of the non-test files of a package that refer to an object accepted by
// match, after resolve has replaced an alias by the type it stands for.
func ListReferences(pkg *packages.Package, match func(types.Object) bool, resolve func(types.Object) types.Object) []Finding {
	var found []Finding
	for ident, object := range pkg.TypesInfo.Uses {
		object = resolve(object)
		if object == nil || object.Pkg() == nil || !match(object) {
			continue
		}
		if strings.HasSuffix(pkg.Fset.Position(ident.Pos()).Filename, "_test.go") {
			continue
		}
		found = append(found, NewFinding(pkg.Fset, ident.Pos(), object.Pkg().Path()+"."+object.Name()))
	}
	return SortFindings(found)
}

// GetDeclaredObject returns the type an alias stands for, and any other object as it is.
func GetDeclaredObject(object types.Object) types.Object {
	name, ok := object.(*types.TypeName)
	if !ok || !name.IsAlias() {
		return object
	}
	if named, ok := types.Unalias(name.Type()).(*types.Named); ok {
		return named.Obj()
	}
	return object
}
