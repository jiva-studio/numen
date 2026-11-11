package container

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// module is the base import path prefix for core packages.
const module = "github.com/jiva-studio/numen/libs/core/kernel/"

func getDriving() map[string]bool {
	return map[string]bool{
		"adapter/cli":                 true,
		"adapter/window/agent":        true,
		"adapter/window/catalog":      true,
		"adapter/window/command":      true,
		"adapter/window/editor":       true,
		"adapter/window/plugin":       true,
		"adapter/window/search":       true,
		"adapter/window/settings":     true,
		"adapter/window/shell":        true,
		"adapter/window/text":         true,
		"adapter/window/vault":        true,
		"adapter/window/vaults":       true,
		"adapter/window/windowstate":  true,
		"adapter/window/workspace":    true,
		"adapter/plugin/hostendpoint": true,
		"adapter/plugin/hostsearch":   true,
		"adapter/plugin/hosttext":     true,
		"adapter/plugin/hostvault":    true,
		"adapter/window/theme":        true,
		"internal/adapterlib/wire":    true,
	}
}

// The adapters depend on the core and the core depends on none of them; an adapter is
// given what it needs and takes no other adapter; assembly happens in one place.
//
// Test files are excluded: a test is external to the package it exercises and
// builds the adapters that stand in for the real ones.
func TestLayerDependenciesPointInward(t *testing.T) {
	var wrong []string
	var read int
	root := ".."
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		from := getPackage(root, path)
		for _, one := range file.Imports {
			to, err := strconv.Unquote(one.Path.Value)
			if err != nil || !strings.HasPrefix(to, module) {
				continue
			}
			read++
			if why := getRefusal(from, strings.TrimPrefix(to, module)); why != "" {
				wrong = append(wrong, from+" reaches "+strings.TrimPrefix(to, module)+": "+why)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one)
	}

	// A walk that reads no edge of this module checks nothing and passes. The count
	// is a lower bound well below the core's edge count; it confirms the walk found
	// the tree, not how large the tree is.
	if read < 100 {
		t.Fatalf("%d edges of this module read: the walk is not reading the core", read)
	}
}

// TestAdaptersDoNotClaimSatisfiedPorts checks that no adapter declares `var _ port.X = …`,
// which duplicates the binding the composition root owns.
// A method returning `port.X` is not such a declaration: where a port opens
// another (`VaultReaders.Open` returns a `VaultReader`), the return type is part
// of the port's signature. The rule rejects only free-standing assertions.
func TestAdaptersDoNotClaimSatisfiedPorts(t *testing.T) {
	var wrong []string
	var read int
	reached := map[string]bool{}
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		pkg := getPackage("..", path)
		if strings.HasSuffix(path, "_test.go") || !isAdapter(pkg) && !isShared(pkg) {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		read++
		reached[getFamily(pkg)] = true
		for _, one := range getClaimedPorts(file) {
			wrong = append(wrong, path+" names port."+one)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one)
	}

	// A walk that reads no file of an adapter checks nothing and passes.
	if read < 50 {
		t.Fatalf("%d files of the adapters read: the walk is not reading them", read)
	}
	if len(reached) < 10 {
		t.Fatalf("%d adapters reached: the walk stops short of the adapters", len(reached))
	}
}

// TestCoreTestsBuildingAdaptersUseExternalPackages checks that a test building an adapter is
// external to the package it exercises, because an adapter built from within the package is in its
// compilation unit. The rules above skip test files for that reason.
// An adapter's own test is not covered: it builds the adapter under test, as does the composition
// root's.
func TestCoreTestsBuildingAdaptersUseExternalPackages(t *testing.T) {
	var wrong []string
	var read int
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return err
		}
		pkg := getPackage("..", path)
		if isAdapter(pkg) || isUnder(pkg, "container") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		read++
		if strings.HasSuffix(file.Name.Name, "_test") {
			return nil
		}
		for _, one := range file.Imports {
			to, err := strconv.Unquote(one.Path.Value)
			if err != nil || !strings.HasPrefix(to, module) {
				continue
			}
			if target := strings.TrimPrefix(to, module); isAdapter(target) {
				wrong = append(wrong, path+" builds "+target+" from inside "+pkg)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one + ": a test that builds an adapter stands outside the package")
	}

	// A walk that read no test of the core is a rule checked against nothing.
	if read < 100 {
		t.Fatalf("%d tests of the core read: the walk is not reading them", read)
	}
}

func isMachinery(pkg string) bool {
	return isUnder(pkg, "container") || strings.HasPrefix(pkg, "internal/test")
}

func isShared(pkg string) bool {
	return strings.HasPrefix(pkg, "internal/adapterlib/")
}

func getMachinePackages() map[string]bool {
	return map[string]bool{
		"os": true, "os/exec": true, "net": true, "net/http": true, "syscall": true,
	}
}

// golang.org/x/sys is the kernel under another name: every package beneath it
// is one operating system's own calls and error numbers.
const systemCalls = "golang.org/x/sys/"

func isMachinePackage(to string) bool {
	return getMachinePackages()[to] || strings.HasPrefix(to, systemCalls)
}

func getFilesystemCalls() map[string]bool {
	return map[string]bool{
		"EvalSymlinks": true, "Glob": true, "Walk": true, "WalkDir": true, "Abs": true,
	}
}

// TestCorePackagesDoNotAccessMachineDirectly checks that a scenario queries a port about the machine and
// never the machine itself.
// Resolving a path through every link is one such query and has no answer for a
// folder passed as a content URI. A file, a socket and a system call are the
// same category. Taking a path apart and joining it is string arithmetic and is allowed.
func TestCorePackagesDoNotAccessMachineDirectly(t *testing.T) {
	var wrong []string
	var read int
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		pkg := getPackage("..", path)
		if strings.HasSuffix(path, "_test.go") || isAdapter(pkg) || isShared(pkg) || isMachinery(pkg) {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		read++
		for _, one := range file.Imports {
			to, err := strconv.Unquote(one.Path.Value)
			if err != nil {
				return err
			}
			if isMachinePackage(to) {
				wrong = append(wrong, path+" is compiled from "+to)
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			pos, is := node.(*ast.SelectorExpr)
			if !is {
				return true
			}
			from, is := pos.X.(*ast.Ident)
			if is && from.Name == "filepath" && getFilesystemCalls()[pos.Sel.Name] {
				wrong = append(wrong, path+" runs filepath."+pos.Sel.Name)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one + ": the core asks a port what is on the machine")
	}

	// A walk that read no file of the core is a rule checked against nothing,
	// and it passes.
	if read < 100 {
		t.Fatalf("%d files of the core read: the walk is not reading it", read)
	}
}

func listTransports() []string {
	return []string{
		"connectrpc.com",
		"database/sql",
		"net/http",
		"github.com/jiva-studio/numen/libs/core/protocol",
	}
}

func isTransport(to string) bool {
	for _, one := range listTransports() {
		if to == one || strings.HasPrefix(to, one+"/") {
			return true
		}
	}
	return false
}

// An inner layer defines the ports for what is outside and imports no transport,
// database driver or schema generator.
func TestInnerLayersDoNotImportTransports(t *testing.T) {
	var wrong []string
	var read int
	for _, root := range []string{"..", "../../../note/domain"} {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			pkg := getPackage("..", path)
			if isAdapter(pkg) || isShared(pkg) || isMachinery(pkg) {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			read++
			for _, one := range file.Imports {
				to, err := strconv.Unquote(one.Path.Value)
				if err != nil {
					return err
				}
				if isTransport(to) {
					wrong = append(wrong, path+" imports "+to)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, one := range wrong {
		t.Error(one + ": the inner layers define ports and import no transport")
	}
	if read < 100 {
		t.Fatalf("%d files of the inner layers read: the walk is not reading them", read)
	}
}

func TestTransportRule_RejectsTransportImports(t *testing.T) {
	var rejected []string
	for _, to := range []string{
		"connectrpc.com/connect",
		"database/sql",
		"database/sql/driver",
		"net/http",
		"net/http/httptest",
		"github.com/jiva-studio/numen/libs/core/protocol",
		"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1",
		"fmt",
		"context",
		module + "port",
	} {
		if isTransport(to) {
			rejected = append(rejected, to)
		}
	}
	want := []string{
		"connectrpc.com/connect",
		"database/sql",
		"database/sql/driver",
		"net/http",
		"net/http/httptest",
		"github.com/jiva-studio/numen/libs/core/protocol",
		"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1",
	}
	if !slices.Equal(rejected, want) {
		t.Errorf("the rule rejected %v, want %v", rejected, want)
	}
}

func getLoggerPackages() map[string]bool {
	return map[string]bool{
		"log": true, "slog": true, "logrus": true, "zap": true,
		"zerolog": true, "logr": true, "glog": true, "klog": true,
	}
}

func isLoggerPackage(to string) bool { return getLoggerPackages()[to[strings.LastIndex(to, "/")+1:]] }

// TestCorePackagesDoNotImportLoggers checks that nothing in the core logs, adapters and
// the composition root included.
// A log is a record for a reader who was not present; the user is in the
// window, and messages are shown there or on the process output stream.
func TestCorePackagesDoNotImportLoggers(t *testing.T) {
	var wrong []string
	var read int
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") ||
			strings.HasSuffix(path, "_test.go") {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		read++
		for _, one := range file.Imports {
			to, err := strconv.Unquote(one.Path.Value)
			if err != nil {
				return err
			}
			if isLoggerPackage(to) {
				wrong = append(wrong, path+" imports "+to)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one + ": nothing here logs — a person is told where they are")
	}

	// A walk that read no file of the core is a rule checked against nothing,
	// and it passes.
	if read < 100 {
		t.Fatalf("%d files of the core read: the walk is not reading it", read)
	}
}

// find both of the standard library's and a third party's, and let through the
// packages whose names only begin the same way.
func TestLoggingRule_RejectsLoggingImports(t *testing.T) {
	var rejected []string
	for _, to := range []string{
		"log", "log/slog", "go.uber.org/zap", "github.com/rs/zerolog",
		"logic", "text/template", module + "domain",
	} {
		if isLoggerPackage(to) {
			rejected = append(rejected, to)
		}
	}
	want := []string{"log", "log/slog", "go.uber.org/zap", "github.com/rs/zerolog"}
	if !slices.Equal(rejected, want) {
		t.Errorf("the rule rejected %v, want %v", rejected, want)
	}
}

func isClockRead(name string) bool { return name == "Now" || name == "Since" || name == "Until" }

func getClockReads(file *ast.File) (reads []string) {
	ast.Inspect(file, func(node ast.Node) bool {
		pos, is := node.(*ast.SelectorExpr)
		if !is {
			return true
		}
		from, is := pos.X.(*ast.Ident)
		if !is || from.Name != "time" {
			return true
		}
		if isClockRead(pos.Sel.Name) {
			reads = append(reads, "time."+pos.Sel.Name)
		}
		return true
	})
	return reads
}

// TestCorePackagesDoNotReadMachineClock checks that the domain has no access to the current time.
// A clock is a port: a scenario that stamps a note or evaluates what is due
// today receives one, and the composition root binds it to this machine's clock.
// A scenario reading the clock itself cannot be exercised at a chosen instant.
func TestCorePackagesDoNotReadMachineClock(t *testing.T) {
	var wrong []string
	var read int
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		pkg := getPackage("..", path)
		if strings.HasSuffix(path, "_test.go") || isAdapter(pkg) || isShared(pkg) || isMachinery(pkg) {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		read++
		for _, one := range getClockReads(file) {
			wrong = append(wrong, path+" runs "+one+": a scenario is handed a clock and reads no other")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one)
	}

	// A walk that read no file of the core is a rule checked against nothing,
	// and it passes.
	if read < 100 {
		t.Fatalf("%d files of the core read: the walk is not reading it", read)
	}
}

func getNumberedTypes() map[string]bool {
	return map[string]bool{
		"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
		"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
		"uintptr": true, "byte": true, "rune": true,
	}
}

// identity records an ID field: containing type, field name, and field type.
type identity struct{ in, field, spelled string }

func getIdentities(file *ast.File) []identity {
	return getFieldsNamed(file, "ID")
}

func getTypeName(pos ast.Expr) string {
	switch target := pos.(type) {
	case *ast.Ident:
		return target.Name
	case *ast.SelectorExpr:
		if from, is := target.X.(*ast.Ident); is {
			return from.Name + "." + target.Sel.Name
		}
	}
	return ""
}

// TestCoreIdentitiesUseDomainTypes checks that every identity in the core is a type of the core's own.
// A row number belongs to the store: it is assigned by whatever counts rows and
// can be ordered and paged, which the entity it names cannot. A core carrying
// one imports the store's numbering into scenario logic, such as a use case
// looping over ascending integer keys.
func TestCoreIdentitiesUseDomainTypes(t *testing.T) {
	var wrong []string
	var read int
	for _, item := range []string{"domain", "port"} {
		dir := getPackageDir(item)
		for _, file := range parseSourceFiles(t, dir) {
			for _, found := range getIdentities(file) {
				read++
				if getNumberedTypes()[found.spelled] {
					wrong = append(wrong, fmt.Sprintf("%s/%s.%s is a %s",
						item, found.in, found.field, found.spelled))
				}
			}
		}
	}
	for _, one := range wrong {
		t.Error(one + ": an identity the core carries is a word of its own")
	}

	// A walk that read no identity is a rule checked against nothing, and it
	// passes.
	if read == 0 {
		t.Fatal("domain/ and port/ name no identity: the walk is not reading them")
	}
}

// has to find the field and the type it is spelled in, and let through the ones
// that are the core's own word and the ones that name no identity at all.
func TestIdentityRule_RejectsIdentityFields(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "identity.go", `package p

type Thing struct {
	ID      VaultID
	ChunkID int64
	OtherID domain.ChunkID
	Rows    int64
	chunkID int64
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	var rejected []string
	for _, one := range getIdentities(file) {
		if getNumberedTypes()[one.spelled] {
			rejected = append(rejected, one.field)
		}
	}
	if !slices.Equal(rejected, []string{"ChunkID"}) {
		t.Errorf("the rule rejected %v, want the one identity spelled as a number", rejected)
	}
}

func getClaimedPorts(file *ast.File) []string {
	var target []string
	ast.Inspect(file, func(node ast.Node) bool {
		decl, is := node.(*ast.GenDecl)
		if !is || decl.Tok != token.VAR {
			return true
		}
		for _, spec := range decl.Specs {
			named, is := spec.(*ast.ValueSpec)
			if !is || !slices.ContainsFunc(named.Names, func(pos *ast.Ident) bool {
				return pos.Name == "_"
			}) {
				continue
			}
			if offset, is := named.Type.(*ast.SelectorExpr); is {
				if from, is := offset.X.(*ast.Ident); is && isPortName(from.Name) {
					target = append(target, offset.Sel.Name)
				}
			}
		}
		return true
	})
	return target
}

// shapes of the claim are one claim, and the walk has to reach the one written
// inside a function; a variable holding a port is left alone.
func TestPortClaimRule_RejectsPortClaims(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "claim.go", `package p

var _ vaultport.VaultReader = (*Reader)(nil)

var _, _ vaultport.VaultWriter = (*Writer)(nil), (*Writer)(nil)

var storePort vaultport.IndexClaim

func mount() {
	var _ noteport.NoteRefs = (*sound)(nil)
	var agentPort agentport.Agent
	_, _ = storePort, agentPort
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"VaultReader", "VaultWriter", "NoteRefs"}
	if got := getClaimedPorts(file); !slices.Equal(got, want) {
		t.Errorf("the rule refuses %v, want %v", got, want)
	}
}

// schema is the generated connect package. An adapter that names one of its handler types serves
// something outside the core, whether over the wire or to another adapter, which makes it a driving
// adapter. An adapter that only builds its messages or clients calls out and is driven.
const schema = wire + "/gen/numen/v1/numenv1connect"

func namesHandler(file *ast.File) bool {
	name := ""
	for _, one := range file.Imports {
		if to, err := strconv.Unquote(one.Path.Value); err == nil && to == schema {
			name = "numenv1connect"
			if one.Name != nil {
				name = one.Name.Name
			}
		}
	}
	if name == "" {
		return false
	}
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		pos, is := node.(*ast.SelectorExpr)
		if !is {
			return true
		}
		if from, is := pos.X.(*ast.Ident); is && from.Name == name && strings.HasSuffix(pos.Sel.Name, "Handler") {
			found = true
		}
		return true
	})
	return found
}

// TestEveryAdapterServingTheSchemaIsDriving ensures adapters serving or mounting
// generated protocol handlers are declared as driving adapters.
func TestEveryAdapterServingTheSchemaIsDriving(t *testing.T) {
	var wrong []string
	var found int
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		pkg := getPackage("..", path)
		if strings.HasSuffix(path, "_test.go") || !isAdapter(pkg) && !isShared(pkg) {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		if !namesHandler(file) {
			return nil
		}
		found++
		if !isDriving(pkg) {
			wrong = append(wrong, getFamily(pkg)+" serves the schema and is named driven")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one)
	}
	if found == 0 {
		t.Fatal("no adapter names the schema: the walk is not reading the adapters")
	}
}

// A port is a dependency the core has on something outside it, and a port with no
// caller is dead. An interface left in port/ after its last caller was removed is
// unused indirection, and the composition root keeps binding an adapter to it.
//
// Test files are read: a port called only from a test counts as used.
func TestPorts_EveryPortTypeIsReferenced(t *testing.T) {
	unasked, err := getUnnamedTypes("..", filepath.Join("..", "port"), true)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range unasked {
		t.Errorf("port.%s is declared and nothing asks for it", one)
	}
}

// TestPortTypesAreReferencedOutsidePort checks that every type a port
// declares is used outside port/; a word nothing outside uses belongs to no
// port and outlives the port that used it.
// The rule does not decide which words belong here: domain versus boundary is
// judged by reading.
func TestPortTypesAreReferencedOutsidePort(t *testing.T) {
	unnamed, err := getUnnamedTypes("..", filepath.Join("..", "port"), false)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range unnamed {
		t.Errorf("port.%s is declared and nothing outside port/ names it", one)
	}
}

// With port/ read as the whole tree, nothing outside it names anything, and the
// rule must return the names below. This confirms the passes above pass on naming
// and not on an empty walk.
func TestPortRule_ReportsUnreferencedPortTypes(t *testing.T) {
	pos := filepath.Join("..", "port")
	unnamed, err := getUnnamedTypes(pos, pos, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range []string{"Embedder", "ChangeSubscriber"} {
		if !slices.Contains(unnamed, one) {
			t.Errorf("nothing outside names port.%s and the rule does not refuse it", one)
		}
	}
}

// TestPortMethodsAreCalledOutsidePort checks that every port method has a
// caller. A method with no caller is never exercised and a defect in it goes unnoticed.
// Test files are not read here, and that asymmetry is the rule: a method called
// only by a test exists to be tested. The two rules above read tests.
func TestPortMethodsAreCalledOutsidePort(t *testing.T) {
	uncalled, err := getUncalledMethods("..", filepath.Join("..", "port"))
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range uncalled {
		t.Errorf("%s is declared and nothing outside a test calls it", one)
	}
}

// Read against port/ as the whole tree, nothing outside it calls anything, and
// the rule has to be restored with the methods below.
func TestPortMethodRule_ReportsUncalledMethods(t *testing.T) {
	pos := filepath.Join("..", "port")
	uncalled, err := getUncalledMethods(pos, pos)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range []string{"NoteRefs.GetNotes"} {
		if !slices.Contains(uncalled, one) {
			t.Errorf("nothing outside calls %s and the rule does not refuse it", one)
		}
	}
}

func getUncalledMethods(root, dir string) ([]string, error) {
	declared, err := getDeclaredMethods(dir)
	if err != nil {
		return nil, err
	}
	if len(declared) < 50 {
		return nil, fmt.Errorf("%s declares %d port methods: the walk is not reading it", dir, len(declared))
	}

	called := map[string]bool{}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		if strings.HasSuffix(path, "_test.go") || isUnder(filepath.Dir(path), filepath.Clean(dir)) {
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
	target, err := readTree(dir)
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
		if isUnder(getPackage(root, path), "port") {
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
			if from, is := pos.X.(*ast.Ident); is && isPortName(from.Name) {
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
	target, err := readTree(dir)
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

func getLayers() map[string]bool {
	return map[string]bool{
		"adapter": true, "container": true, "domain": true, "port": true, "usecase": true,
	}
}

// TestNoPackageIsImportedBelowItsLayer verifies that packages do not use layer names as import aliases.
func TestNoPackageIsImportedBelowItsLayer(t *testing.T) {
	var wrong []string
	var read, aliased int
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		read++
		for _, one := range file.Imports {
			if one.Name == nil {
				continue
			}
			aliased++
			if !getLayers()[one.Name.Name] {
				continue
			}
			target, err := strconv.Unquote(one.Path.Value)
			if err != nil {
				return err
			}
			wrong = append(wrong, path+" names "+target+" "+one.Name.Name)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one)
	}

	// A walk that read no file, or one that met no alias, is a rule checked
	// against nothing, and it passes: this rule has only aliases to read.
	if read < 100 {
		t.Fatalf("%d files of the core read: the walk is not reading it", read)
	}
	if aliased < 10 {
		t.Fatalf("%d imports carry a name of their own: the walk is reading no alias", aliased)
	}
}

// TestLayerAliasRule_RejectsLayerNamedAliases verifies rejection of layer names as import aliases.
func TestLayerAliasRule_RejectsLayerNamedAliases(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "alias.go", `package p

import (
	"maps"
	usecase "`+module+`usecase/note"
	adapter "`+module+`adapter/index"
	vaults "`+module+`usecase/vault"
	"`+module+`domain"
	pathpkg "path"
)
`, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	var rejected []string
	for _, one := range file.Imports {
		if one.Name != nil && getLayers()[one.Name.Name] {
			rejected = append(rejected, one.Name.Name)
		}
	}
	if want := []string{"usecase", "adapter"}; !slices.Equal(rejected, want) {
		t.Errorf("the rule rejected %v, want %v", rejected, want)
	}
}

func getPackage(root, path string) string {
	target, err := filepath.Rel(root, filepath.Dir(path))
	if err != nil {
		return path
	}
	return filepath.ToSlash(target)
}

func isAdapter(pkg string) bool {
	return strings.HasPrefix(pkg, "adapter/") ||
		strings.HasPrefix(pkg, "internal/adapter/")
}

func getRefusal(from, to string) string {
	switch {
	case isAdapter(from) || isShared(from):
		if isUnder(to, "container") {
			return "an adapter is given what it needs and assembles nothing"
		}
		if isAdapter(to) && !isSibling(from, to) {
			return "an adapter is given what it needs and takes no other adapter"
		}
		if strings.HasPrefix(to, "usecase/") && !isDriving(from) {
			return "a driven adapter is behind a port and runs no scenario"
		}
		if isUnder(to, "port") && isDriving(from) {
			return "a driving adapter calls the use case layer and imports no port directly"
		}
	case isUnder(from, "container"):
		if !isAssembled(to) {
			return "the composition root assembles the core and does none of its work"
		}
	case isUnder(from, "domain"):
		if !isUnder(to, "domain") {
			return "a domain package imports the standard library, the notes domain and the domain packages, and nothing else of the core"
		}
	// Fixtures build the real adapters, and only a test is compiled from them.
	case strings.HasPrefix(from, "internal/test"):
	default:
		if isAdapter(to) || isUnder(to, "container") {
			return "the core reaches no adapter and nothing that assembles one"
		}
	}
	return ""
}

func isAmong(these []string, pkg string) bool {
	for _, one := range these {
		if pkg == one || strings.HasPrefix(pkg, one+"/") {
			return true
		}
	}
	return false
}

func isAssembled(to string) bool {
	return isUnder(to, "container") || isUnder(to, "domain") || isUnder(to, "port") ||
		isAdapter(to) || strings.HasPrefix(to, "usecase/")
}

func isSibling(from, to string) bool { return getFamily(from) == getFamily(to) }

func listGrouping() []string {
	return []string{"adapter", "internal/adapter", "adapter/window", "internal/adapterlib"}
}

func getFamily(pkg string) string {
	depth := 2
	for _, one := range listGrouping() {
		if strings.HasPrefix(pkg, one+"/") {
			depth = max(depth, strings.Count(one, "/")+2)
		}
	}
	target := strings.Split(pkg, "/")
	if len(target) < depth {
		return pkg
	}
	return strings.Join(target[:depth], "/")
}

// holds took the name of one it does not and was given that name's baseline
// entry, and a package holding what is true of a note reached one that holds
// work.
func TestLayerRule_RefusesForbiddenEdges(t *testing.T) {
	for _, one := range []struct {
		from, to  string
		isRefused bool
	}{
		// The window is given what it needs and assembles nothing, and one
		// settings file is the union of sections each adapter owns.
		{"adapter/window/editor", "container", true},
		{"adapter/settings", "internal/adapter/filesystem", true},

		// The folder holding one adapter to a window is not an adapter, so the
		// two windows are two adapters and neither is given the other.
		{"adapter/window/editor", "adapter/window/shell", true},
		{"adapter/window/shell", "adapter/window/editor", true},
		{"adapter/window/editor/pages", "adapter/window/editor", false},

		// A driven adapter runs no scenario and takes no other adapter. Which of the
		// two an adapter is does not depend on its folder: the compiler-restricted
		// folder here also serves the schema.
		{"adapter/index", "usecase/note", true},
		{"internal/adapter/filesystem", "usecase/note", true},
		{"adapter/window/theme", "usecase/note", false},
		{"adapter/window/theme", "internal/adapter/filesystem", true},
		{"internal/adapter/themefiles", "usecase/note", true},
		{"internal/adapter/themefiles", "adapter/window/theme", true},
		{"adapter/window/theme", "internal/adapter/themefiles", true},
		{"usecase/theme", "internal/adapter/themefiles", true},
		{"adapter/index/chunk", "adapter/index", false},

		// The core imports no adapter and no package that assembles one.
		{"usecase/note", "internal/adapter/trash", true},
		{"usecase/note", "container", true},
		{"usecase/note", "container/vault", true},
		{"usecase/note", "containerized", false},
		{"usecase/note", "port", false},
		{"internal/adapterlib/wire", "adapter/window/editor", true},
		{"internal/adapterlib/wire", "container", true},
		{"internal/adapterlib/wire", "internal/adapter/trash", true},
		{"internal/adapterlib/wire", "usecase/note", false},
		{"internal/adapterlib/onnxruntime", "usecase/note", true},
		{"internal/adapterlib/onnxruntime", "internal/adapter/filesystem", true},
		{"internal/adapterlib/onnxruntime", "port", false},

		// A domain package of the core imports no other layer of it, and every layer
		// above it may import the domain.
		{"domain/plugin", "port", true},
		{"domain/plugin", "port/pluginport", true},
		{"domain/plugin", "usecase/note", true},
		{"domain/plugin", "adapter/index", true},
		{"domain/plugin", "internal/adapter/trash", true},
		{"domain/plugin", "internal/text", true},
		{"domain/plugin", "container", true},
		{"domain/plugin", "domain/settings", false},
		{"domain/settings/sub", "domain/settings", false},
		{"container", "domain/plugin", false},
		{"usecase/note", "domain/plugin", false},
		{"port/pluginport", "domain/plugin", false},
		{"adapter/window/editor", "domain/plugin", false},
		{"internal/adapter/plugin", "domain/plugin", false},

		// Fixtures build the real adapters.
		{"internal/testsupport", "adapter/index", false},
		{"internal/testonly", "container", false},

		// The composition root does none of the core's work.
		{"container", "internal/adapterlib/wire", true},
		{"container", "internal/text", true},
		{"container", "adapter/index", false},
		{"container", "usecase/run", false},
	} {
		why := getRefusal(one.from, one.to)
		if one.isRefused && why == "" {
			t.Errorf("%s reaches %s and is not rejected", one.from, one.to)
		}
		if !one.isRefused && why != "" {
			t.Errorf("%s reaches %s and is rejected: %s", one.from, one.to, why)
		}
	}
}

// Reading the tree is what the rules are checked against, so the tree has to be
// where this expects it.
func TestLayerDirectoriesExist(t *testing.T) {
	for _, item := range []string{"domain", "port", "usecase", "adapter", "internal/adapter", "internal/adapterlib"} {
		if _, err := os.Stat(getPackageDir(item)); err != nil {
			t.Fatalf("%s: %v", item, err)
		}
	}
	// getPackageDir reads "domain" as the notes domain; the domain packages of the core are read here.
	if _, err := os.Stat(filepath.Join("..", "domain")); err != nil {
		t.Fatal(err)
	}
}

func getInstants(file *ast.File) []identity {
	return getFieldsNamed(file, "Time")
}

// TestCoreInstantsUseTimeType checks that an instant in domain/ and port/ is a time.Time.
// A second count and a millisecond count are both integers, so using one where
// the other is expected compiles and is off by a factor of a thousand; the zero
// of either is a valid instant in 1970.
func TestCoreInstantsUseTimeType(t *testing.T) {
	var wrong []string
	var read int
	for _, item := range []string{"domain", "port"} {
		dir := getPackageDir(item)
		for _, file := range parseSourceFiles(t, dir) {
			for _, found := range getInstants(file) {
				read++
				if found.spelled != "time.Time" {
					wrong = append(wrong, fmt.Sprintf("%s/%s.%s is a %s",
						item, found.in, found.field, found.spelled))
				}
			}
		}
	}
	for _, one := range wrong {
		t.Error(one + ": an instant the core carries is a time.Time")
	}

	// A walk that read no instant is a rule checked against nothing, and it
	// passes.
	if read == 0 {
		t.Fatal("domain/ and port/ name no instant: the walk is not reading them")
	}
}

// finds the field and the type it is written in, and lets through the one that
// is a time and the ones that name no instant at all.
func TestInstantRule_RejectsTimeFields(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "instant.go", `package p

import "time"

type Reading struct {
	ModTime  time.Time
	SeenTime int64
	Pages    int
	runTime  int64
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	var rejected []string
	for _, one := range getInstants(file) {
		if one.spelled != "time.Time" {
			rejected = append(rejected, one.field)
		}
	}
	if !slices.Equal(rejected, []string{"SeenTime"}) {
		t.Errorf("rejected %v, want SeenTime", rejected)
	}
}

func getMethodIdentities(file *ast.File) []identity {
	var found []identity
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
						found = append(found, identity{
							spec.Name.Name + "." + method.Names[0].Name,
							name.Name,
							getTypeName(one.Type),
						})
					}
				}
			}
		}
		return true
	})
	return found
}

// The same rule applied to port signatures. An identity passed through a port as
// a number brings the store's numbering into the core's language, and a signature
// is where the crossing occurs.
func TestPortMethodIdentitiesUseDomainTypes(t *testing.T) {
	var wrong []string
	var read int
	dir := filepath.Join("..", "port")
	for _, file := range parseSourceFiles(t, dir) {
		for _, found := range getMethodIdentities(file) {
			read++
			if getNumberedTypes()[found.spelled] {
				wrong = append(wrong, fmt.Sprintf("port/%s(%s %s)",
					found.in, found.field, found.spelled))
			}
		}
	}
	for _, one := range wrong {
		t.Error(one + ": an identity a port asks for is a word of the core's own")
	}

	// A walk that read no identity is a rule checked against nothing, and it
	// passes.
	if read == 0 {
		t.Fatal("port/ asks for no identity by name: the walk is not reading it")
	}
}

// checks parameters and results alike, accepts the type that is the core's own,
// and ignores unnamed parameters.
func TestIdentityRule_RejectsIdentitySignatureParams(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "asking.go", `package p

import "context"

type Store interface {
	Read(ctx context.Context, vaultID domain.VaultID, chunkID int64) error
	Write(ctx context.Context, rowID int) (itemID domain.ItemID, err error)
	Count(context.Context, int64) int
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	var rejected []string
	for _, one := range getMethodIdentities(file) {
		if getNumberedTypes()[one.spelled] {
			rejected = append(rejected, one.field)
		}
	}
	if !slices.Equal(rejected, []string{"chunkID", "rowID"}) {
		t.Errorf("the rule rejected %v, want the two identities spelled as numbers", rejected)
	}
}

// the clock reads, and lets through timers, the arithmetic over an instant and
// the constructors that read no clock.
func TestClockRule_RejectsClockReads(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "clock.go", `package p

import "time"

func run(origin time.Time) {
	_ = time.Now()
	_ = time.Since(origin)
	_ = time.Until(origin)
	<-time.After(time.Second)
	time.AfterFunc(time.Second, func() {})
	_ = time.NewTimer(time.Second)
	_ = time.NewTicker(time.Second)
	_ = time.Tick(time.Second)
	_ = origin.Add(time.Second)
	_ = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_ = time.Duration(3)
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	if reads, want := getClockReads(file), []string{"time.Now", "time.Since", "time.Until"}; !slices.Equal(reads, want) {
		t.Errorf("the rule refuses the reads %v, want %v", reads, want)
	}
}

// TestContainerDoesNotDowncastPluginHost ensures container code interacts with
// plugin supervisor solely through the capabilities of container.PluginHost, never downcasting
// to concrete *plugin.Host.
func TestContainerDoesNotDowncastPluginHost(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		node, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(node, func(n ast.Node) bool {
			ta, ok := n.(*ast.TypeAssertExpr)
			if !ok {
				return true
			}
			star, ok := ta.Type.(*ast.StarExpr)
			if !ok {
				return true
			}
			sel, ok := star.X.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			if pkg.Name == "plugin" && sel.Sel.Name == "Host" {
				t.Errorf("%s: downcast to *plugin.Host is forbidden; use container.PluginHost", file)
			}
			return true
		})
	}
}

func getPackageDir(pos string) string {
	if pos == "domain" {
		return filepath.Join("..", "..", "..", "note", "domain")
	}
	return filepath.Join("..", pos)
}

// treeEntry is one file of a folder tree, named by its path under the folder.
type treeEntry struct{ name string }

// Name returns the path of the file under the folder that was read.
func (e treeEntry) Name() string { return e.name }

// IsDir reports false: a tree entry is a file.
func (e treeEntry) IsDir() bool { return false }

func readTree(dir string) ([]treeEntry, error) {
	var found []treeEntry
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		found = append(found, treeEntry{name: filepath.ToSlash(rel)})
		return nil
	})
	return found, err
}

func isUnder(pkg, folder string) bool {
	return pkg == folder || strings.HasPrefix(pkg, folder+"/")
}

func isPortName(name string) bool {
	return slices.Contains(getPortNames(), name)
}

func getPortNames() []string {
	entries, err := os.ReadDir(getPackageDir("port"))
	if err != nil {
		return nil
	}
	var names []string
	for _, one := range entries {
		if one.IsDir() {
			names = append(names, one.Name())
		}
	}
	return names
}

func getSerializationImports() map[string]bool {
	return map[string]bool{
		"encoding/json":              true,
		"encoding/xml":               true,
		"gopkg.in/yaml.v3":           true,
		"github.com/BurntSushi/toml": true,
		"google.golang.org/protobuf": true,
	}
}

func listSerializationTags() []string {
	return []string{"json:", "yaml:", "sql:", "db:", "xml:", "protobuf:"}
}

// TestNoInnerLayerImportsSerializationOrUsesFormatTags verifies that port/, usecase/, and domain/
// import no wire/storage format packages and carry no serialization struct tags.
func TestNoInnerLayerImportsSerializationOrUsesFormatTags(t *testing.T) {
	for _, root := range []string{"../domain", "../port", "../usecase", "../../../note/domain"} {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				return err
			}
			if root != "../domain" {
				for _, imp := range file.Imports {
					to, err := strconv.Unquote(imp.Path.Value)
					if err != nil {
						continue
					}
					if getSerializationImports()[to] || strings.HasPrefix(to, "google.golang.org/protobuf/") {
						t.Errorf("%s imports %s: inner layers do not parse wire or storage formats", path, to)
					}
				}
			}
			ast.Inspect(file, func(n ast.Node) bool {
				spec, ok := n.(*ast.TypeSpec)
				if !ok {
					return true
				}
				st, ok := spec.Type.(*ast.StructType)
				if !ok || st.Fields == nil {
					return true
				}
				for _, field := range st.Fields.List {
					if field.Tag == nil {
						continue
					}
					for _, tag := range listSerializationTags() {
						if strings.Contains(field.Tag.Value, tag) {
							fieldName := "<embedded>"
							if len(field.Names) > 0 {
								fieldName = field.Names[0].Name
							}
							t.Errorf("%s %s.%s has serialization tag %s: format tags belong on adapter DTOs", path, spec.Name.Name, fieldName, field.Tag.Value)
						}
					}
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// TestUseCasesImportNoOSFilepathOrFS verifies that usecase/ never imports path/filepath or io/fs.
func TestUseCasesImportNoOSFilepathOrFS(t *testing.T) {
	err := filepath.WalkDir("../usecase", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range file.Imports {
			to, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				continue
			}
			if to == "path/filepath" || to == "io/fs" {
				t.Errorf("%s imports %s: use cases operate on slash-separated vault paths and domain sentinels, not OS filepath or io/fs", path, to)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestNoStructCarriesAContext verifies that no struct field across Go modules holds context.Context.
func TestNoStructCarriesAContext(t *testing.T) {
	roots := []string{"..", "../../../note", "../../../plugin/sdk-go", "../../../../plugins", "../../../../apps/desktop"}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			if strings.HasSuffix(path, ".pb.go") || strings.HasSuffix(path, ".connect.go") {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				return err
			}
			ast.Inspect(file, func(n ast.Node) bool {
				spec, ok := n.(*ast.TypeSpec)
				if !ok {
					return true
				}
				st, ok := spec.Type.(*ast.StructType)
				if !ok || st.Fields == nil {
					return true
				}
				for _, field := range st.Fields.List {
					if getTypeName(field.Type) == "context.Context" {
						fieldName := "<embedded>"
						if len(field.Names) > 0 {
							fieldName = field.Names[0].Name
						}
						t.Errorf("%s %s.%s holds context.Context in a struct field (pass context per method call instead)", path, spec.Name.Name, fieldName)
					}
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// TestNothingOutsideAdaptersReadsTheMachineClock verifies this across libs/note, plugins, and apps/desktop.
func TestNothingOutsideAdaptersReadsTheMachineClock(t *testing.T) {
	roots := []string{"../../../note", "../../../../plugins", "../../../../apps/desktop"}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			slash := filepath.ToSlash(path)
			if strings.Contains(slash, "/cmd/") || strings.Contains(slash, "/internal/adapter/") || strings.Contains(slash, "/testsupport/") {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				return err
			}
			for _, one := range getClockReads(file) {
				t.Errorf("%s runs %s: inject a clock port instead of reading the machine clock", path, one)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func getFieldsNamed(file *ast.File, suffix string) []identity {
	var found []identity
	ast.Inspect(file, func(node ast.Node) bool {
		spec, is := node.(*ast.TypeSpec)
		if !is {
			return true
		}
		target, is := spec.Type.(*ast.StructType)
		if !is {
			return true
		}
		for _, one := range target.Fields.List {
			for _, name := range one.Names {
				if name.IsExported() && strings.HasSuffix(name.Name, suffix) {
					found = append(found, identity{spec.Name.Name, name.Name, getTypeName(one.Type)})
				}
			}
		}
		return true
	})
	return found
}

func parseSourceFiles(t *testing.T, dir string) []*ast.File {
	t.Helper()
	entries, err := readTree(dir)
	if err != nil {
		t.Fatal(err)
	}
	var files []*ast.File
	for _, one := range entries {
		if one.IsDir() || !strings.HasSuffix(one.Name(), ".go") || strings.HasSuffix(one.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, one.Name()), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, file)
	}
	return files
}

func isDriving(pkg string) bool {
	return isAmong(slices.Collect(maps.Keys(getDriving())), pkg)
}

// The rule counts a package that names a handler type or constructor of the connect package, and
// lets through one that only builds messages or calls a client.
func TestHandlerRule_SeparatesServingFromCalling(t *testing.T) {
	for _, tc := range []struct {
		what, body string
		isServing  bool
	}{
		{"a handler constructor", `numenv1connect.NewNoteServiceHandler(h)`, true},
		{"a handler type", `var _ numenv1connect.NoteServiceHandler`, true},
		{"a client constructor", `numenv1connect.NewNoteServiceClient(nil, "")`, false},
		{"a service name", `_ = numenv1connect.NoteServiceName`, false},
	} {
		source := "package p\n\nimport \"" + schema + "\"\n\nfunc f(h any) {\n\t" + tc.body + "\n}\n"
		file, err := parser.ParseFile(token.NewFileSet(), "p.go", source, 0)
		if err != nil {
			t.Fatal(err)
		}
		if got := namesHandler(file); got != tc.isServing {
			t.Errorf("%s: serving is %v, want %v", tc.what, got, tc.isServing)
		}
	}
}
