package container

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// module is the base import path prefix for core packages.
const module = "github.com/jiva-studio/numen/libs/core/"

// driving lists driving adapters that expose external interfaces and call use cases.
var driving = map[string]bool{
	"adapter/cli":              true,
	"adapter/window/editor":    true,
	"adapter/window/shell":     true,
	"internal/adapter/plugin":  true,
	"internal/adapter/theme":   true,
	"internal/adapterlib/wire": true,
}

// The adapters depend on the core and the core depends on none of them; an adapter is
// given what it needs and takes no other adapter; assembly happens in one place.
//
// Test files are excluded: a test is external to the package it exercises and
// builds the adapters that stand in for the real ones.
func TestTheLayersAreWhatTheyAre(t *testing.T) {
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

// A port is named after the need and an adapter after the technology; the
// composition root owns the binding between them. An adapter that declares
// `var _ port.X = …` duplicates that binding.
//
// A method returning `port.X` is not such a declaration. Where a port opens
// another (`VaultReaders.Open` returns a `VaultReader`), the return type is part
// of the port's signature, and an adapter bound to the first must declare the
// second. The rule rejects only free-standing assertions.
func TestNoAdapterNamesThePortItSatisfies(t *testing.T) {
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
		reached[family(pkg)] = true
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

	// A walk that reads no file of an adapter checks nothing and passes. Every
	// adapter is named, so the count is compared against all of them.
	if read < 50 {
		t.Fatalf("%d files of the adapters read: the walk is not reading them", read)
	}
	for _, one := range public {
		if !reached["adapter/"+one] {
			t.Errorf("no file of adapter/%s was read: the rule stops at that adapter", one)
		}
	}
	for _, one := range internalAdapters {
		if !reached["internal/adapter/"+one] {
			t.Errorf("no file of internal/adapter/%s was read: the rule stops at that adapter", one)
		}
	}
}

// A test that builds an adapter is external to the package it exercises. The
// rules above skip test files for that reason: an adapter built from within the
// package is in its compilation unit when the tests build, and every other
// scenario in the core is tested from a package_test.
//
// An adapter's own test is not covered by this: it builds the adapter under test,
// as does the composition root's.
func TestATestOfTheCoreBuildingAnAdapterExistsOutsideIt(t *testing.T) {
	var wrong []string
	var read int
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return err
		}
		pkg := getPackage("..", path)
		if isAdapter(pkg) || pkg == "container" {
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

// isMachinery reports whether a package is the composition root or a test fixture.
func isMachinery(pkg string) bool {
	return pkg == "container" || strings.HasPrefix(pkg, "internal/test")
}

// isShared reports whether a package is a shared adapter library.
func isShared(pkg string) bool {
	return strings.HasPrefix(pkg, "internal/adapterlib/")
}

// theMachine are the packages that expose the machine and the world beyond it:
// the process's files and children, sockets, and the kernel. A scenario obtains
// each of these through a port.
//
// net/url is excluded. Parsing and building addresses is string arithmetic and
// dials nothing.
var theMachine = map[string]bool{
	"os": true, "os/exec": true, "net": true, "net/http": true, "syscall": true,
}

// golang.org/x/sys is the kernel under another name: every package beneath it
// is one operating system's own calls and error numbers.
const systemCalls = "golang.org/x/sys/"

// isTheMachine answers whether an import is the machine.
func isTheMachine(to string) bool { return theMachine[to] || strings.HasPrefix(to, systemCalls) }

// looking are the functions of path/filepath that are not path arithmetic.
// Each queries the filesystem, and Abs resolves against the folder the process
// started in, which a scenario must not depend on.
var looking = map[string]bool{
	"EvalSymlinks": true, "Glob": true, "Walk": true, "WalkDir": true, "Abs": true,
}

// A scenario queries a port about the machine and never queries the machine itself.
// Resolving a path through every link is one such query, and it has no answer
// for a folder passed as a content URI: the disk it would read is not the one
// holding the file. A file, a socket and a system call are the same category of query.
//
// Taking a path apart and joining it is string arithmetic and is allowed. The
// distinction is whether the answer is on the disk.
func TestNothingOfTheCoreReachesTheMachine(t *testing.T) {
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
			if isTheMachine(to) {
				wrong = append(wrong, path+" is compiled from "+to)
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			pos, is := node.(*ast.SelectorExpr)
			if !is {
				return true
			}
			from, is := pos.X.(*ast.Ident)
			if is && from.Name == "filepath" && looking[pos.Sel.Name] {
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

// transports are the wire protocols, database drivers, schema generators and network
// handlers. An inner layer defines the ports for them and imports none of them.
var transports = []string{
	"connectrpc.com",
	"database/sql",
	"net/http",
	"github.com/jiva-studio/numen/libs/protocol",
}

// isTransport answers whether an import is a transport, a database driver or a wire schema.
func isTransport(to string) bool {
	for _, one := range transports {
		if to == one || strings.HasPrefix(to, one+"/") {
			return true
		}
	}
	return false
}

// An inner layer defines the ports for what is outside and imports no transport,
// database driver or schema generator.
func TestNoInnerLayerImportsATransport(t *testing.T) {
	var wrong []string
	var read int
	for _, root := range []string{"..", "../../notes/domain"} {
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
		"github.com/jiva-studio/numen/libs/protocol",
		"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1",
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
		"github.com/jiva-studio/numen/libs/protocol",
		"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1",
	}
	if !slices.Equal(rejected, want) {
		t.Errorf("the rule rejected %v, want %v", rejected, want)
	}
}

// logging are the words a package that keeps a record is named by, read off the
// last element of an import path. The standard library holds two of them, and
// the rest are the libraries reached for when one of those will not do.
var logging = map[string]bool{
	"log": true, "slog": true, "logrus": true, "zap": true,
	"zerolog": true, "logr": true, "glog": true, "klog": true,
}

// logs answers whether an import is a logger.
func logs(to string) bool { return logging[to[strings.LastIndex(to, "/")+1:]] }

// Nothing here logs. A log is a record for a reader who was not present; the
// application's user is in the window, and messages are shown there or on the
// output stream of the process. A bound logger would write to neither.
//
// The rule covers the whole core, so the adapters and the composition root are
// read too.
func TestNothingOfTheCoreLogs(t *testing.T) {
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
			if logs(to) {
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
		if logs(to) {
			rejected = append(rejected, to)
		}
	}
	want := []string{"log", "log/slog", "go.uber.org/zap", "github.com/rs/zerolog"}
	if !slices.Equal(rejected, want) {
		t.Errorf("the rule rejected %v, want %v", rejected, want)
	}
}

// telling are the functions of time that read the machine's clock, and waiting
// are the ones that arm the machine's timers. The rest of the package is
// arithmetic over a given instant.
var (
	telling = map[string]bool{"Now": true, "Since": true}
	waiting = map[string]bool{
		"After": true, "AfterFunc": true, "NewTimer": true, "NewTicker": true, "Tick": true,
	}
)

// getClockReads are the calls in a file that read the machine's clock or arm its
// timers, each written time.Name. The local name of the import is not read.
func getClockReads(file *ast.File) (reads, arms []string) {
	ast.Inspect(file, func(node ast.Node) bool {
		pos, is := node.(*ast.SelectorExpr)
		if !is {
			return true
		}
		from, is := pos.X.(*ast.Ident)
		if !is || from.Name != "time" {
			return true
		}
		if telling[pos.Sel.Name] {
			reads = append(reads, "time."+pos.Sel.Name)
		}
		if waiting[pos.Sel.Name] {
			arms = append(arms, "time."+pos.Sel.Name)
		}
		return true
	})
	return reads, arms
}

// The domain has no access to the current time. A clock is a port: a scenario
// that stamps a note or evaluates what is due today receives one, and the
// composition root binds it to this machine's clock.
//
// A scenario that reads the clock itself cannot be exercised at a chosen
// instant. Much of the work here depends on time: the day a plugin schedules
// something for, and the identifier a note is stamped with.
func TestNothingOfTheCoreReadsTheMachinesClock(t *testing.T) {
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
		reads, arms := getClockReads(file)
		for _, one := range reads {
			wrong = append(wrong, path+" runs "+one+": a scenario is handed a clock and reads no other")
		}
		if len(arms) > 0 {
			wrong = append(wrong, path+" runs "+arms[0]+": the core arms no timer of the machine's")
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

// numbered are the types a machine's own counter is spelled in.
var numbered = map[string]bool{
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"uintptr": true, "byte": true, "rune": true,
}

// identity records an ID field: containing type, field name, and field type.
type identity struct{ in, field, spelled string }

// identities are the exported struct fields of one file whose names end in ID.
func identities(file *ast.File) []identity {
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
				if name.IsExported() && strings.HasSuffix(name.Name, "ID") {
					found = append(found, identity{spec.Name.Name, name.Name, getTypeName(one.Type)})
				}
			}
		}
		return true
	})
	return found
}

// getTypeName is a type as it is written, and empty for one this rule reads
// nothing into.
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

// An identity in the core is a type of the core's own. A row number belongs to the
// store: it identifies a row of a table, is assigned by whatever counts rows, and
// can be ordered and paged where the entity it names cannot. A core carrying one
// imports the store's numbering into its language and into scenario logic, such
// as a use case looping over ascending integer keys.
//
// Every identity is its own type; this rule enforces that.
func TestTheCoresIdentitiesAreItsOwn(t *testing.T) {
	var wrong []string
	var read int
	for _, item := range []string{"domain", "port"} {
		dir := getPackageDir(item)
		target, err := readTree(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, one := range target {
			if one.IsDir() || !strings.HasSuffix(one.Name(), ".go") ||
				strings.HasSuffix(one.Name(), "_test.go") {
				continue
			}
			file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, one.Name()), nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			for _, found := range identities(file) {
				read++
				if numbered[found.spelled] {
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
	for _, one := range identities(file) {
		if numbered[one.spelled] {
			rejected = append(rejected, one.field)
		}
	}
	if !slices.Equal(rejected, []string{"ChunkID"}) {
		t.Errorf("the rule rejected %v, want the one identity spelled as a number", rejected)
	}
}

// getClaimedPorts are the ports a file declares itself to answer, by the blank
// name.
//
// A blank standing among other names is the same claim as one standing alone,
// and a declaration inside a function is the same claim as one beside the
// package's own. A variable that holds a port is not one: it is given the
// adapter, and names what it was given.
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
	var _ noteport.NoteQueries = (*sound)(nil)
	var agentPort agentport.Agent
	_, _ = storePort, agentPort
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"VaultReader", "VaultWriter", "NoteQueries"}
	if got := getClaimedPorts(file); !slices.Equal(got, want) {
		t.Errorf("the rule refuses %v, want %v", got, want)
	}
}

// schema is the generated messages. An adapter that accepts and returns them
// serves something outside the core, whether over the wire or to another adapter,
// which makes it a driving adapter.
//
// The handler interface it satisfies does not show this: an adapter never names
// the interface it implements, so only the messages identify it.
const schema = wire + "/gen/numen/v1"

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
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, one := range file.Imports {
			to, err := strconv.Unquote(one.Path.Value)
			if err != nil || to != schema {
				continue
			}
			found++
			if !driving[family(pkg)] {
				wrong = append(wrong, family(pkg)+" serves the schema and is named driven")
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

// A port is named in the words of its own domain, and those words are declared
// next to it: the identity of a model, a page of audio, a row of the settings
// file. A word that nothing outside port/ uses belongs to no port, and remains
// after the port that used it is removed.
//
// This rule does not decide which words belong here. Whether a type is domain or
// boundary is judged by reading: what the index holds is domain, and what an
// adapter is configured with is boundary.
func TestEveryTypePortDeclaresIsNamedSomewhereElse(t *testing.T) {
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
	for _, one := range []string{"Vector", "ErrorHandler"} {
		if !slices.Contains(unnamed, one) {
			t.Errorf("nothing outside names port.%s and the rule does not refuse it", one)
		}
	}
}

// A port is consumed as a whole, and each of its methods is one operation the
// core can call. A method with no caller is never exercised, adapters keep
// implementing it, and a defect in it goes unnoticed.
//
// Test files are not read here, and that asymmetry is the rule. A port used only
// by a test is still a port, so the two rules above read tests. A method called
// only by a test exists to be tested: its only caller is the code written to
// check it.
func TestEveryPortMethodIsCalledSomewhereElse(t *testing.T) {
	uncalled, err := getUncalledMethods("..", filepath.Join("..", "port"))
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range uncalled {
		t.Errorf("%s is declared and nothing outside a test calls it", one)
	}
}

// Read against port/ as the whole tree, nothing outside it calls anything, and
// the rule has to come back with the methods below.
func TestPortMethodRule_ReportsUncalledMethods(t *testing.T) {
	pos := filepath.Join("..", "port")
	uncalled, err := getUncalledMethods(pos, pos)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range []string{"NoteQueries.GetNotes"} {
		if !slices.Contains(uncalled, one) {
			t.Errorf("nothing outside calls %s and the rule does not refuse it", one)
		}
	}
}

// getUncalledMethods returns the methods declared by port/'s interfaces that no
// non-test file outside port/ names as a selector.
//
// A selector is the form of a method call site regardless of the receiver's
// type, so matching is on the name alone: an interface the core calls through
// a variable never writes port.X at the call site.
func getUncalledMethods(root, dir string) ([]string, error) {
	declared, err := methodsIn(dir)
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

// methodsIn are the methods the exported interfaces of a folder declare, each
// keyed as Interface.Method and giving the method's own name. An embedded
// interface is the other one's declaration and is read there.
func methodsIn(dir string) (map[string]string, error) {
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

// getUnnamedTypes are the exported types port/ declares that no file of the
// tree outside it names as port.X. onlyPorts reads the interfaces alone.
//
// The tests are read: a type a test alone still names is named.
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

// getDeclaredTypes returns the exported types a folder's files declare, each
// flagged as an interface or not.
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

// layers are the folders the tree is laid out in.
var layers = map[string]bool{
	"adapter": true, "container": true, "domain": true, "port": true, "usecase": true,
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
			if !layers[one.Name.Name] {
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
		if one.Name != nil && layers[one.Name.Name] {
			rejected = append(rejected, one.Name.Name)
		}
	}
	if want := []string{"usecase", "adapter"}; !slices.Equal(rejected, want) {
		t.Errorf("the rule rejected %v, want %v", rejected, want)
	}
}

// getPackage is the package a file belongs to, as the rules name it.
func getPackage(root, path string) string {
	target, err := filepath.Rel(root, filepath.Dir(path))
	if err != nil {
		return path
	}
	return filepath.ToSlash(target)
}

// isAdapter reports whether a package is an adapter.
func isAdapter(pkg string) bool {
	return strings.HasPrefix(pkg, "adapter/") ||
		strings.HasPrefix(pkg, "internal/adapter/")
}

// getRefusal returns why one package may not import another.
func getRefusal(from, to string) string {
	switch {
	case isAdapter(from) || isShared(from):
		if to == "container" {
			return "an adapter is given what it needs and assembles nothing"
		}
		if isAdapter(to) && !sibling(from, to) {
			return "an adapter is given what it needs and takes no other adapter"
		}
		if strings.HasPrefix(to, "usecase/") && !driving[family(from)] {
			return "a driven adapter stands behind a port and runs no scenario"
		}
		if isUnder(to, "port") && driving[family(from)] {
			return "a driving adapter calls the use case layer and imports no port directly"
		}
	case from == "container":
		if !isAssembled(to) {
			return "the composition root assembles the core and does none of its work"
		}
	// Fixtures build the real adapters, and only a test is compiled from them.
	case strings.HasPrefix(from, "internal/test"):
	default:
		if isAdapter(to) || to == "container" {
			return "the core reaches no adapter and nothing that assembles one"
		}
	}
	return ""
}

// isAmong reports whether a package is one of these or is placed under one.
func isAmong(these []string, pkg string) bool {
	for _, one := range these {
		if pkg == one || strings.HasPrefix(pkg, one+"/") {
			return true
		}
	}
	return false
}

// isAssembled reports whether a package is one the composition root assembles:
// an adapter, a scenario, or one of the two language packages they are named in.
func isAssembled(to string) bool {
	return to == "container" || to == "domain" || isUnder(to, "port") ||
		isAdapter(to) || strings.HasPrefix(to, "usecase/")
}

// sibling reports whether two packages belong to one adapter: its own folder and
// every package under it, which share the settings section they are built from.
func sibling(from, to string) bool { return family(from) == family(to) }

// grouping are the folders that contain adapters and are not adapters themselves:
// adapter/ itself, the compiler-restricted internal folder, and the folder
// holding one adapter per window. A package directly under any of them is an adapter.
var grouping = []string{"adapter", "internal/adapter", "adapter/window", "internal/adapterlib"}

// family is the adapter a package belongs to: the package directly under its
// grouping folder, and every package beneath that.
func family(pkg string) string {
	depth := 2
	for _, one := range grouping {
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

// getAdaptersIn are the adapters in one folder, named under it. A folder that
// groups adapters is not an adapter, so what is placed under it is read in its
// place.
func getAdaptersIn(dir, pos string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var found []string
	for _, one := range entries {
		if !one.IsDir() {
			continue
		}
		below := pos + "/" + one.Name()
		if !slices.Contains(grouping, below) {
			found = append(found, one.Name())
			continue
		}
		deeper, err := getAdaptersIn(filepath.Join(dir, one.Name()), below)
		if err != nil {
			return nil, err
		}
		for _, name := range deeper {
			found = append(found, one.Name()+"/"+name)
		}
	}
	return found, nil
}

// public are the adapters an application names for itself: the four it serves
// something through, and the two it composes or configures.
var public = []string{
	"cli", "index", "settings",
	"window/editor", "window/shell",
}

// The core's surface is these adapters and no others. A driven adapter that
// nothing outside composes lives under internal/adapter, which the compiler
// restricts, so binding it to a port stays this package's responsibility.
func TestTheCoresPublicAdaptersAreTheseAndNoOthers(t *testing.T) {
	found, err := getAdaptersIn(filepath.Join("..", "adapter"), "adapter")
	if err != nil {
		t.Fatal(err)
	}
	there := make(map[string]bool, len(found))
	for _, one := range found {
		there[one] = true
		if !isAmong(public, one) {
			t.Errorf("adapter/%s is public and nothing outside composes it", one)
		}
	}
	for _, one := range public {
		if !there[one] {
			t.Errorf("adapter/%s is named here and is not there", one)
		}
	}
}

// internalAdapters lists adapters internal to this module.
var internalAdapters = []string{
	"appstate", "filesystem", "models", "onnxruntime", "plugin", "provision", "sharedmimeinfo",
	"theme", "trash",
}

// The adapters under internal/ are these and no others.
func TestInternalAdapters_MatchDeclaredList(t *testing.T) {
	found, err := getAdaptersIn(filepath.Join("..", "internal", "adapter"), "internal/adapter")
	if err != nil {
		t.Fatal(err)
	}
	there := make(map[string]bool, len(found))
	for _, one := range found {
		there[one] = true
		if !isAmong(internalAdapters, one) {
			t.Errorf("internal/adapter/%s stands here and is named nowhere", one)
		}
	}
	for _, one := range internalAdapters {
		if !there[one] {
			t.Errorf("internal/adapter/%s is named here and is not there", one)
		}
	}
}

// holds took the name of one it does not and was given that name's baseline
// entry, and a package holding what is true of a note reached one that holds
// work.
func TestWhatTheRulesRefuse(t *testing.T) {
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
		{"internal/adapter/theme", "usecase/note", false},
		{"internal/adapter/theme", "internal/adapter/filesystem", true},
		{"internal/adapter/theme/presets", "internal/adapter/theme", false},
		{"adapter/index/chunk", "adapter/index", false},

		// The core imports no adapter and no package that assembles one.
		{"usecase/note", "internal/adapter/trash", true},
		{"usecase/note", "container", true},
		{"usecase/note", "port", false},
		{"internal/adapterlib/wire", "adapter/window/editor", true},
		{"internal/adapterlib/wire", "container", true},
		{"internal/adapterlib/wire", "internal/adapter/trash", true},
		{"internal/adapterlib/wire", "usecase/note", false},
		{"internal/adapterlib/onnxruntime", "usecase/note", true},
		{"internal/adapterlib/onnxruntime", "internal/adapter/filesystem", true},
		{"internal/adapterlib/onnxruntime", "port", false},

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
func TestTheTreeIsWhereTheLayersAreRead(t *testing.T) {
	for _, item := range []string{"domain", "port", "usecase", "adapter", "internal/adapter", "internal/adapterlib"} {
		if _, err := os.Stat(getPackageDir(item)); err != nil {
			t.Fatalf("%s: %v", item, err)
		}
	}
}

// getInstants are the fields of a struct whose names end in Time, by the type
// each is written in.
func getInstants(file *ast.File) []identity {
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
				if name.IsExported() && strings.HasSuffix(name.Name, "Time") {
					found = append(found, identity{spec.Name.Name, name.Name, getTypeName(one.Type)})
				}
			}
		}
		return true
	})
	return found
}

// An instant in the core is a time.Time. A second count and a millisecond count
// are both integers, so using one where the other is expected compiles and is
// off by a factor of a thousand; the zero of either is a valid instant in 1970,
// and the name does not say which clock it came from.
//
// The rule applies to domain/ and port/, where the core declares its own types
// and what it requires of the outside.
func TestEveryInstantTheCoreCarriesIsATime(t *testing.T) {
	var wrong []string
	var read int
	for _, item := range []string{"domain", "port"} {
		dir := getPackageDir(item)
		target, err := readTree(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, one := range target {
			if one.IsDir() || !strings.HasSuffix(one.Name(), ".go") ||
				strings.HasSuffix(one.Name(), "_test.go") {
				continue
			}
			file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, one.Name()), nil, 0)
			if err != nil {
				t.Fatal(err)
			}
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

// getMethodIdentities extracts ID parameters and return types from interface methods.
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
func TestTheIdentitiesAPortAsksForAreTheCoresOwn(t *testing.T) {
	var wrong []string
	var read int
	dir := filepath.Join("..", "port")
	target, err := readTree(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range target {
		if one.IsDir() || !strings.HasSuffix(one.Name(), ".go") ||
			strings.HasSuffix(one.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, one.Name()), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, found := range getMethodIdentities(file) {
			read++
			if numbered[found.spelled] {
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
		if numbered[one.spelled] {
			rejected = append(rejected, one.field)
		}
	}
	if !slices.Equal(rejected, []string{"chunkID", "rowID"}) {
		t.Errorf("the rule rejected %v, want the two identities spelled as numbers", rejected)
	}
}

// the clock read and each timer armed, and lets through the arithmetic over an
// instant and the constructors that read no clock.
func TestClockRule_RejectsClockReads(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "clock.go", `package p

import "time"

func run(origin time.Time) {
	_ = time.Now()
	_ = time.Since(origin)
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
	reads, arms := getClockReads(file)
	if want := []string{"time.Now", "time.Since"}; !slices.Equal(reads, want) {
		t.Errorf("the rule refuses the reads %v, want %v", reads, want)
	}
	want := []string{"time.After", "time.AfterFunc", "time.NewTimer", "time.NewTicker", "time.Tick"}
	if !slices.Equal(arms, want) {
		t.Errorf("the rule refuses the timers %v, want %v", arms, want)
	}
}

// TestContainerDoesNotDowncastPluginHost ensures container code interacts with
// plugin supervisor solely through port.PluginManager interface, never downcasting
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
				t.Errorf("%s: downcast to *plugin.Host is forbidden; use port.PluginManager", file)
			}
			return true
		})
	}
}

// getPackageDir returns the filesystem path for a package name.
func getPackageDir(pos string) string {
	if pos == "domain" {
		return filepath.Join("..", "..", "notes", "domain")
	}
	return filepath.Join("..", pos)
}

// treeEntry is one file of a folder tree, named by its path under the folder.
type treeEntry struct{ name string }

// Name returns the path of the file under the folder that was read.
func (e treeEntry) Name() string { return e.name }

// IsDir reports false: a tree entry is a file.
func (e treeEntry) IsDir() bool { return false }

// readTree reads every file under a folder and the folders beneath it.
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

// isUnder reports whether a package is a folder or is within it.
func isUnder(pkg, folder string) bool {
	return pkg == folder || strings.HasPrefix(pkg, folder+"/")
}

// isPortName reports whether a name is one under which a file imports a package
// of port/: the folder name each of them is declared as.
func isPortName(name string) bool {
	return slices.Contains(getPortNames(), name)
}

// getPortNames are the names of the packages under port/.
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

var serializationImports = map[string]bool{
	"encoding/json":              true,
	"encoding/xml":               true,
	"gopkg.in/yaml.v3":           true,
	"github.com/BurntSushi/toml": true,
	"google.golang.org/protobuf": true,
}

var serializationTags = []string{"json:", "yaml:", "sql:", "db:", "xml:", "protobuf:"}

// TestNoInnerLayerImportsSerializationOrUsesFormatTags verifies that port/, usecase/, and domain/
// import no wire/storage format packages and carry no serialization struct tags.
func TestNoInnerLayerImportsSerializationOrUsesFormatTags(t *testing.T) {
	for _, root := range []string{"../port", "../usecase", "../../notes/domain"} {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				return err
			}
			for _, imp := range file.Imports {
				to, err := strconv.Unquote(imp.Path.Value)
				if err != nil {
					continue
				}
				if serializationImports[to] || strings.HasPrefix(to, "google.golang.org/protobuf/") {
					t.Errorf("%s imports %s: inner layers do not parse wire or storage formats", path, to)
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
					for _, tag := range serializationTags {
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
	roots := []string{"..", "../../notes", "../../plugin-sdk-go", "../../../plugins", "../../../apps/desktop"}
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

// TestNothingOutsideAdaptersReadsClockOrRandom verifies this across libs/notes, plugins, and apps/desktop.
func TestNothingOutsideAdaptersReadsClockOrRandom(t *testing.T) {
	roots := []string{"../../notes", "../../../plugins", "../../../apps/desktop"}
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
			reads, arms := getClockReads(file)
			for _, one := range reads {
				t.Errorf("%s runs %s: inject a clock port instead of reading the machine clock", path, one)
			}
			for _, one := range arms {
				t.Errorf("%s runs %s: inject a clock/timer port instead of arming machine timers", path, one)
			}
			ast.Inspect(file, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if id, ok := sel.X.(*ast.Ident); ok && id.Name == "rand" && (sel.Sel.Name == "Read" || strings.HasPrefix(sel.Sel.Name, "Int")) {
					t.Errorf("%s runs rand.%s: inject an entropy source instead of calling global rand", path, sel.Sel.Name)
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
