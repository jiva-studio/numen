package container

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
)

// clockReaders lists the functions of package time that read the machine clock.
var clockReaders = []string{"Now", "Since", "Until"}

// clockRead is a reference to a machine clock reader found in the source.
type clockRead struct {
	// position is the file:line of the reference.
	position string
	// name is the referenced function, such as time.Now.
	name string
	// isSanctioned reports whether the reference is inside the body of a method
	// Now() time.Time whose receiver implements a clock port.
	isSanctioned bool
}

// isTimeTime reports whether t is the type time.Time.
func isTimeTime(t types.Type) bool {
	named, ok := t.(*types.Named)
	return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == "time" && named.Obj().Name() == "Time"
}

// isClockSignature reports whether sig is func() time.Time.
func isClockSignature(sig *types.Signature) bool {
	return sig.Params().Len() == 0 && sig.Results().Len() == 1 && isTimeTime(sig.Results().At(0).Type())
}

// getClockPorts returns the interfaces declared in non-test code of pkgs whose only method is
// Now() time.Time.
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

// isClockMethod reports whether decl is a method Now() time.Time whose receiver implements one of ports.
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

// isClockReader reports whether object is time.Now, time.Since or time.Until.
func isClockReader(object types.Object) bool {
	function, ok := object.(*types.Func)
	if !ok || function.Pkg() == nil || function.Pkg().Path() != "time" || !slices.Contains(clockReaders, function.Name()) {
		return false
	}
	sig, ok := function.Type().(*types.Signature)
	return ok && sig.Recv() == nil
}

// clockWalk collects the clock reads of one package.
type clockWalk struct {
	pkg   *packages.Package
	ports []*types.Interface
	// enclosing holds, for each function around the visited node, whether it is a clock method.
	enclosing []bool
	reads     []clockRead
	visited      map[string]bool
}

// visit records the clock reads under node.
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

// descend visits body as the innermost function.
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

// getMachineClockReads returns every reference to time.Now, time.Since and time.Until in the
// non-generated files of pkgs, test files included. A reference is sanctioned when the innermost
// enclosing function is a method Now() time.Time implementing a clock port.
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
	slices.SortFunc(reads, func(a, b clockRead) int { return strings.Compare(a.position, b.position) })
	return reads
}

// loadPackages loads the packages matching patterns under dir with syntax and types, tests included.
func loadPackages(t *testing.T, dir string, env []string, patterns ...string) []*packages.Package {
	t.Helper()
	config := &packages.Config{
		Dir:   dir,
		Env:   append(os.Environ(), env...),
		Tests: true,
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports |
			packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
	}
	loaded, err := packages.Load(config, patterns...)
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

// listWorkspaceModules returns the directory of every module in go.work.
func listWorkspaceModules(t *testing.T) []string {
	t.Helper()
	command := exec.CommandContext(t.Context(), "go", "work", "edit", "-json")
	command.Dir = numenModulesDir
	output, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	var work struct {
		Use []struct{ DiskPath string }
	}
	if err := json.Unmarshal(output, &work); err != nil {
		t.Fatal(err)
	}
	dirs := make([]string, len(work.Use))
	for i, one := range work.Use {
		abs, err := filepath.Abs(filepath.Join(numenModulesDir, one.DiskPath))
		if err != nil {
			t.Fatal(err)
		}
		dirs[i] = abs
	}
	return dirs
}

// The machine clock is read in one place: the body of a method Now() time.Time on a type that
// implements a clock port. Every other reference to time.Now, time.Since or time.Until in
// non-generated Go, tests included, takes the time from an injected clock.
func TestTheMachineClockIsReadOnlyByAClockPortImplementation(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks every module of the workspace")
	}
	var reads []clockRead
	loaded := 0
	for _, dir := range listWorkspaceModules(t) {
		pkgs := loadPackages(t, dir, nil, "./...")
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
		t.Errorf("%s references %s outside the Now method of a clock port implementation", one.position, one.name)
	}

	// A load that finds no clock implementation checks nothing. The system clocks of the core
	// and of the plugin SDK read the machine clock.
	if loaded < 100 || sanctioned < 2 {
		t.Fatalf("%d packages loaded and %d clock implementations found: the rule is checked against nothing", loaded, sanctioned)
	}
}

// writeFixture writes the files into a new module named fixture and returns its directory.
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

// The rule rejects each reference outside a clock port implementation and accepts the one inside.
func TestClockPortRule_RejectsReadsOutsideAClockImplementation(t *testing.T) {
	dir := writeFixture(t, map[string]string{
		"port.go": `package fixture

import "time"

type Clock interface{ Now() time.Time }

type System struct{}

func (System) Now() time.Time { return time.Now() }
`,
		"violations.go": `package fixture

import "time"

type Stamp struct{}

func (Stamp) Now(zone string) time.Time { return time.Now() }

type Wrapped struct{}

func (Wrapped) Now() time.Time {
	read := func() time.Time { return time.Now() }
	return read()
}

func Plain() time.Time { return time.Now() }

func Elapsed(origin time.Time) time.Duration { return time.Since(origin) }

func Remaining(origin time.Time) time.Duration { return time.Until(origin) }

var source = time.Now
`,
		"violations_test.go": `package fixture

import "time"

func readInTest() time.Time { return time.Now() }
`,
		"generated.go": `// Code generated by a tool. DO NOT EDIT.

package fixture

import "time"

func generated() time.Time { return time.Now() }
`,
	})
	reads := getMachineClockReads(loadPackages(t, dir, []string{"GOWORK=off"}, "./..."))
	var rejected, accepted []string
	for _, one := range reads {
		line := filepath.Base(one.position) + " " + one.name
		if one.isSanctioned {
			accepted = append(accepted, line)
		} else {
			rejected = append(rejected, line)
		}
	}
	wantRejected := []string{
		"violations.go:7 time.Now",
		"violations.go:12 time.Now",
		"violations.go:16 time.Now",
		"violations.go:18 time.Since",
		"violations.go:20 time.Until",
		"violations.go:22 time.Now",
		"violations_test.go:5 time.Now",
	}
	slices.Sort(rejected)
	slices.Sort(wantRejected)
	if !slices.Equal(rejected, wantRejected) {
		t.Errorf("rejected %v, want %v", rejected, wantRejected)
	}
	if want := []string{"port.go:9 time.Now"}; !slices.Equal(accepted, want) {
		t.Errorf("accepted %v, want %v", accepted, want)
	}
}

// A method Now() time.Time without a clock port in the loaded packages is not a clock implementation.
func TestClockPortRule_RejectsAClockMethodWithoutAPort(t *testing.T) {
	dir := writeFixture(t, map[string]string{
		"lone.go": `package fixture

import "time"

type Lone struct{}

func (Lone) Now() time.Time { return time.Now() }
`,
	})
	reads := getMachineClockReads(loadPackages(t, dir, []string{"GOWORK=off"}, "./..."))
	if len(reads) != 1 || reads[0].isSanctioned {
		t.Errorf("reads %v, want one unsanctioned read", reads)
	}
}
