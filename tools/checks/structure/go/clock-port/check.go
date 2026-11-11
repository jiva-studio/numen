package clockport

import (
	"encoding/json"
	"go/ast"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"golang.org/x/tools/go/packages"
	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

func listClockReaders() []string {
	return []string{"Now", "Since", "Until"}
}

// clockRead is a reference to a machine clock reader found in the source.
type clockRead struct {
	position     string
	name         string
	isSanctioned bool
}

func isTimeTime(t types.Type) bool {
	named, ok := t.(*types.Named)
	return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == "time" && named.Obj().Name() == "Time"
}

func isClockSignature(sig *types.Signature) bool {
	return sig.Params().Len() == 0 && sig.Results().Len() == 1 && isTimeTime(sig.Results().At(0).Type())
}

func getClockPorts(pkgs []*packages.Package) []*types.Interface {
	var ports []*types.Interface
	visited := map[string]bool{}
	for _, pkg := range pkgs {
		if pkg.Types == nil {
			continue
		}
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			object, ok := scope.Lookup(name).(*types.TypeName)
			if !ok {
				continue
			}
			position := pkg.Fset.Position(object.Pos())
			if strings.HasSuffix(position.Filename, "_test.go") || visited[position.String()] {
				continue
			}
			iface, ok := object.Type().Underlying().(*types.Interface)
			if !ok || iface.NumMethods() != 1 || iface.NumEmbeddeds() != 0 || iface.Method(0).Name() != "Now" {
				continue
			}
			sig, ok := iface.Method(0).Type().(*types.Signature)
			if !ok || !isClockSignature(sig) {
				continue
			}
			visited[position.String()] = true
			ports = append(ports, iface)
		}
	}
	return ports
}

func isClockMethod(decl *ast.FuncDecl, info *types.Info, ports []*types.Interface) bool {
	if decl.Recv == nil || decl.Name.Name != "Now" || decl.Body == nil {
		return false
	}
	function, ok := info.Defs[decl.Name].(*types.Func)
	if !ok {
		return false
	}
	sig, ok := function.Type().(*types.Signature)
	if !ok || !isClockSignature(sig) {
		return false
	}
	receiver := sig.Recv().Type()
	if pointer, isPointer := receiver.(*types.Pointer); isPointer {
		receiver = pointer.Elem()
	}
	for _, port := range ports {
		if types.Implements(receiver, port) || types.Implements(types.NewPointer(receiver), port) {
			return true
		}
	}
	return false
}

func isClockReader(object types.Object) bool {
	function, ok := object.(*types.Func)
	if !ok || function.Pkg() == nil || function.Pkg().Path() != "time" || !slices.Contains(listClockReaders(), function.Name()) {
		return false
	}
	sig, ok := function.Type().(*types.Signature)
	return ok && sig.Recv() == nil
}

// clockWalk collects the clock reads of one package.
type clockWalk struct {
	pkg       *packages.Package
	ports     []*types.Interface
	enclosing []bool
	reads     []clockRead
	visited   map[string]bool
}

func (w *clockWalk) visit(node ast.Node) bool {
	switch current := node.(type) {
	case *ast.FuncDecl:
		if current.Body != nil {
			w.descend(current.Body, isClockMethod(current, w.pkg.TypesInfo, w.ports))
		}
		return false
	case *ast.FuncLit:
		w.descend(current.Body, false)
		return false
	case *ast.Ident:
		object := w.pkg.TypesInfo.Uses[current]
		if object == nil || !isClockReader(object) {
			return true
		}
		position := w.pkg.Fset.Position(current.Pos())
		if w.visited[position.String()] {
			return true
		}
		w.visited[position.String()] = true
		w.reads = append(w.reads, clockRead{
			position:     filepath.ToSlash(position.Filename) + ":" + strconv.Itoa(position.Line),
			name:         "time." + object.Name(),
			isSanctioned: len(w.enclosing) > 0 && w.enclosing[len(w.enclosing)-1],
		})
	}
	return true
}

func (w *clockWalk) descend(body *ast.BlockStmt, isClock bool) {
	w.enclosing = append(w.enclosing, isClock)
	ast.Inspect(body, func(inner ast.Node) bool {
		if inner == nil {
			return true
		}
		return w.visit(inner)
	})
	w.enclosing = w.enclosing[:len(w.enclosing)-1]
}

func getMachineClockReads(pkgs []*packages.Package) []clockRead {
	ports := getClockPorts(pkgs)
	visited := map[string]bool{}
	var reads []clockRead
	for _, pkg := range pkgs {
		if pkg.TypesInfo == nil {
			continue
		}
		walk := &clockWalk{pkg: pkg, ports: ports, enclosing: nil, reads: nil, visited: visited}
		for _, file := range pkg.Syntax {
			if ast.IsGenerated(file) {
				continue
			}
			ast.Inspect(file, func(node ast.Node) bool {
				if node == nil {
					return true
				}
				return walk.visit(node)
			})
		}
		reads = append(reads, walk.reads...)
	}
	slices.SortFunc(reads, func(a, b clockRead) int { return strings.Compare(a.Position, b.Position) })
	return reads
}

func checkMachineClockIsReadOnlyByAClockPortImplementation(t *testing.T, workspace load.WorkspacePackages) {
	t.Helper()
	var reads []clockRead
	loaded := 0
	for _, dir := range load.ListWorkspaceModules(t) {
		pkgs := workspace.ListModule(t, dir)
		for _, pkg := range pkgs {
			for _, one := range pkg.Errors {
				t.Errorf("%s does not load: %v", pkg.ID, one)
			}
		}
		loaded += len(pkgs)
		reads = append(reads, getMachineClockReads(pkgs)...)
	}
	sanctioned := 0
	for _, one := range reads {
		if one.isSanctioned {
			sanctioned++
			continue
		}
		t.Errorf("%s references %s outside the Now method of a clock port implementation", one.Position, one.Name)
	}

	// A load that finds no clock implementation checks nothing. The system clocks of the core
	// and of the plugin SDK read the machine clock.
	if loaded < 100 || sanctioned < 2 {
		t.Fatalf("%d packages loaded and %d clock implementations found: the rule is checked against nothing", loaded, sanctioned)
	}
}

func writeFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	files["go.mod"] = "module fixture\n\ngo 1.26\n"
	for name, text := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}
