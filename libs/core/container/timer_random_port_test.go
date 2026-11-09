package container

import (
	"go/ast"
	"go/types"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

// listTimers returns the functions of package time that start a timer or wait on the machine's clock.
func listTimers() []string {
	return []string{"After", "AfterFunc", "NewTimer", "NewTicker", "Tick", "Sleep"}
}

// listRandomSources returns the packages whose every object reads a source of randomness.
func listRandomSources() []string {
	return []string{"crypto/rand", "math/rand", "math/rand/v2"}
}

func isTimerOrRandomRead(object types.Object) bool {
	if object.Pkg() == nil {
		return false
	}
	if slices.Contains(listRandomSources(), object.Pkg().Path()) {
		return true
	}
	function, ok := object.(*types.Func)
	if !ok || object.Pkg().Path() != "time" || !slices.Contains(listTimers(), function.Name()) {
		return false
	}
	return function.Signature().Recv() == nil
}

// getDeclaredPorts returns the interfaces with methods that the code of the workspace declares: those of the
// non-test files of the packages, and the exported ones of the workspace packages they import.
func getDeclaredPorts(pkgs []*packages.Package) []*types.Interface {
	visited := map[*types.TypeName]bool{}
	var ports []*types.Interface
	add := func(object types.Object) {
		name, ok := object.(*types.TypeName)
		if !ok || name.IsAlias() || visited[name] {
			return
		}
		iface, ok := name.Type().Underlying().(*types.Interface)
		if !ok || !iface.IsMethodSet() || iface.NumMethods() == 0 {
			return
		}
		visited[name] = true
		ports = append(ports, iface)
	}
	seen := map[*types.Package]bool{}
	var walkImports func(*types.Package)
	walkImports = func(pkg *types.Package) {
		if seen[pkg] {
			return
		}
		seen[pkg] = true
		if isWithin(pkg.Path(), strings.TrimSuffix(numenRoot, "/")) {
			for _, name := range pkg.Scope().Names() {
				if object := pkg.Scope().Lookup(name); object.Exported() {
					add(object)
				}
			}
		}
		for _, imported := range pkg.Imports() {
			walkImports(imported)
		}
	}
	for _, pkg := range listProductionPackages(pkgs) {
		for _, name := range pkg.Types.Scope().Names() {
			add(pkg.Types.Scope().Lookup(name))
		}
		seen[pkg.Types] = true
		for _, imported := range pkg.Types.Imports() {
			walkImports(imported)
		}
	}
	return ports
}

// isPortMethod reports whether a declaration is a method of a type that implements a declared port of which
// the method is one.
func isPortMethod(decl *ast.FuncDecl, info *types.Info, ports []*types.Interface) bool {
	if decl.Recv == nil || decl.Body == nil {
		return false
	}
	function, ok := info.Defs[decl.Name].(*types.Func)
	if !ok {
		return false
	}
	receiver := function.Signature().Recv().Type()
	if pointer, isPointer := receiver.(*types.Pointer); isPointer {
		receiver = pointer.Elem()
	}
	for _, port := range ports {
		if !hasMethod(port, function.Name()) {
			continue
		}
		if types.Implements(receiver, port) || types.Implements(types.NewPointer(receiver), port) {
			return true
		}
	}
	return false
}

func hasMethod(iface *types.Interface, name string) bool {
	for method := range iface.Methods() {
		if method.Name() == name {
			return true
		}
	}
	return false
}

func getTimerAndRandomReads(pkgs []*packages.Package) []clockRead {
	ports := getDeclaredPorts(pkgs)
	return getMachineReads(pkgs, isTimerOrRandomRead, func(decl *ast.FuncDecl, info *types.Info) bool {
		return isPortMethod(decl, info, ports)
	})
}

func checkTimersAndRandomnessAreReadOnlyInAPortImplementation(t *testing.T, workspace workspacePackages) {
	t.Helper()
	var reads []clockRead
	loaded := 0
	for _, dir := range listWorkspaceModules(t) {
		pkgs := workspace.listModule(t, dir)
		loaded += len(pkgs)
		reads = append(reads, getTimerAndRandomReads(pkgs)...)
	}
	for _, one := range reads {
		if !one.isSanctioned {
			t.Errorf("%s references %s outside a method of a type that implements a declared port", one.position, one.name)
		}
	}

	// A load that finds no timer and no source of randomness checks nothing: the application starts timers.
	if loaded < 100 || len(reads) == 0 {
		t.Fatalf("%d packages loaded and %d references to timers and randomness found: the rule is checked against nothing", loaded, len(reads))
	}
}

// The rule accepts a timer or a random source used in a port method of an implementation, and rejects it
// in a plain function, a closure, a method that is no port method, a type that implements no port, and a test.
func TestTimerAndRandomPortRule_RejectsReadsOutsideAPortImplementation(t *testing.T) {
	dir := writeFixture(t, map[string]string{
		"port.go": `package fixture

import (
	"crypto/rand"
	mathrand "math/rand/v2"
	"time"
)

type Waiter interface{ Wait(d time.Duration) }

type Dice interface{ Roll() int }

type Machine struct{}

func (Machine) Wait(d time.Duration) { time.Sleep(d) }

func (*Machine) Roll() int { return mathrand.IntN(6) }

func (Machine) Other() { time.Sleep(time.Second) }

type Lone struct{}

func (Lone) Wait() { <-time.After(time.Second) }

func Plain() []byte {
	key := make([]byte, 8)
	_, _ = rand.Read(key)
	return key
}

func Later(run func()) { time.AfterFunc(time.Second, run) }

type Wrapped struct{}

func (Wrapped) Roll() int {
	roll := func() int { return mathrand.IntN(6) }
	return roll()
}

var _ = time.Now
`,
		"port_test.go": `package fixture

import "time"

func pause() { time.Sleep(time.Millisecond) }
`,
	})
	reads := getTimerAndRandomReads(loadPackages(t, dir, []string{"GOWORK=off"}, "./..."))
	var rejected, accepted []string
	for _, one := range reads {
		line := filepath.Base(one.position) + " " + one.name
		if one.isSanctioned {
			accepted = append(accepted, line)
		} else {
			rejected = append(rejected, line)
		}
	}
	wantAccepted := []string{"port.go:15 time.Sleep", "port.go:17 math/rand/v2.IntN"}
	wantRejected := []string{
		"port.go:19 time.Sleep",
		"port.go:23 time.After",
		"port.go:27 crypto/rand.Read",
		"port.go:31 time.AfterFunc",
		"port.go:36 math/rand/v2.IntN",
		"port_test.go:5 time.Sleep",
	}
	slices.Sort(accepted)
	slices.Sort(rejected)
	if !slices.Equal(accepted, wantAccepted) {
		t.Errorf("accepted %v, want %v", accepted, wantAccepted)
	}
	if !slices.Equal(rejected, wantRejected) {
		t.Errorf("rejected %v, want %v", rejected, wantRejected)
	}
}
