package load

import (
	"go/types"

	"golang.org/x/tools/go/packages"
)

// PortInterface is an interface with methods that a package under port/ declares.
type PortInterface struct {
	Name  string
	Iface *types.Interface
}

func IsInLayer(path, root, layer string) bool {
	return IsWithin(path, root+"/"+layer)
}

func ListPortInterfaces(pkgs []*packages.Package, root string) []PortInterface {
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
