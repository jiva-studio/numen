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

// machineWalk collects the references of one package to what reads the machine: the clock, a timer or a
// source of randomness. A reference is sanctioned when the function declaration that directly encloses it,
// not through a closure, is one the rule accepts.
type machineWalk struct {
	pkg          *packages.Package
	isRead       func(types.Object) bool
	isSanctioned func(*ast.FuncDecl, *types.Info) bool
	enclosing    []bool
	reads        []clockRead
	visited      map[string]bool
}

func (w *machineWalk) visit(node ast.Node) bool {
	switch current := node.(type) {
	case *ast.FuncDecl:
		if current.Body != nil {
			w.descend(current.Body, w.isSanctioned(current, w.pkg.TypesInfo))
		}
		return false
	case *ast.FuncLit:
		w.descend(current.Body, false)
		return false
	case *ast.Ident:
		object := w.pkg.TypesInfo.Uses[current]
		if object == nil || !w.isRead(object) {
			return true
		}
		position := w.pkg.Fset.Position(current.Pos())
		if w.visited[position.String()] {
			return true
		}
		w.visited[position.String()] = true
		w.reads = append(w.reads, clockRead{
			position:     filepath.ToSlash(position.Filename) + ":" + strconv.Itoa(position.Line),
			name:         object.Pkg().Path() + "." + object.Name(),
			isSanctioned: len(w.enclosing) > 0 && w.enclosing[len(w.enclosing)-1],
		})
	}
	return true
}

func (w *machineWalk) descend(body *ast.BlockStmt, isSanctioned bool) {
	w.enclosing = append(w.enclosing, isSanctioned)
	ast.Inspect(body, func(inner ast.Node) bool {
		if inner == nil {
			return true
		}
		return w.visit(inner)
	})
	w.enclosing = w.enclosing[:len(w.enclosing)-1]
}

func getMachineReads(pkgs []*packages.Package, isRead func(types.Object) bool, isSanctioned func(*ast.FuncDecl, *types.Info) bool) []clockRead {
	visited := map[string]bool{}
	var reads []clockRead
	for _, pkg := range pkgs {
		if pkg.TypesInfo == nil {
			continue
		}
		walk := &machineWalk{pkg: pkg, isRead: isRead, isSanctioned: isSanctioned, enclosing: nil, reads: nil, visited: visited}
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

func getMachineClockReads(pkgs []*packages.Package) []clockRead {
	ports := getClockPorts(pkgs)
	return getMachineReads(pkgs, isClockReader, func(decl *ast.FuncDecl, info *types.Info) bool {
		return isClockMethod(decl, info, ports)
	})
}


func loadPackages(t *testing.T, dir string, env []string, patterns ...string) []*packages.Package {
	t.Helper()
	loaded, err := packages.Load(newTypedConfig(dir, env), patterns...)
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func newTypedConfig(dir string, env []string) *packages.Config {
	return &packages.Config{
		Dir:   dir,
		Env:   append(os.Environ(), env...),
		Tests: true,
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports |
			packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
	}
}

func listWorkspaceModules(t *testing.T) []string {
	t.Helper()
	command := exec.CommandContext(t.Context(), "go", "work", "edit", "-json")
	command.Dir = repoRootDir
	output, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	var work struct {
		Use []struct {
			DiskPath string `json:"DiskPath"`
		} `json:"Use"`
	}
	if err := json.Unmarshal(output, &work); err != nil {
		t.Fatal(err)
	}
	dirs := make([]string, len(work.Use))
	for i, one := range work.Use {
		abs, err := filepath.Abs(filepath.Join(repoRootDir, one.DiskPath))
		if err != nil {
			t.Fatal(err)
		}
		dirs[i] = abs
	}
	return dirs
}

func checkMachineClockIsReadOnlyByAClockPortImplementation(t *testing.T, workspace workspacePackages) {
	t.Helper()
	var reads []clockRead
	loaded := 0
	for _, dir := range listWorkspaceModules(t) {
		pkgs := workspace.listModule(t, dir)
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
