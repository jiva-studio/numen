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
const module = "github.com/jiva-studio/numen/modules/libs/core/"

// driving lists driving adapters that expose external interfaces and call use cases.
var driving = map[string]bool{
	"adapter/cli":               true,
	"adapter/window/editor":     true,
	"adapter/window/flashcards": true,
	"internal/adapter/plugin":   true,
	"internal/adapter/theme":    true,
	"internal/adapterlib/wire":  true,
}

// pure are the packages holding what is true of a note, of a
// highlight, a recognised page, a transcript or a proofread line, and the
// arithmetic over them. Nothing they answer waits on a disk, a database or a
// clock, and the way that is held is that a pure package reaches only another
// pure one: naming no port and no scenario is not enough on its own, because a
// sibling of the core carries goroutines, channels and a schema.
var pure = []string{
	"internal/highlight", "internal/ocr", "internal/transcript",
}

// The core is reached by the adapters and reaches none of them, an adapter is
// given what it needs and not taking another, and what is assembled is
// assembled in one place.
//
// A test file is left out: a test stands outside the package it exercises and
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

	// A walk that read no edge of this module is a rule checked against
	// nothing, and it passes. The count is a floor well under what the core
	// holds, so it says the walk found the tree and not how big the tree is.
	if read < 100 {
		t.Fatalf("%d edges of this module read: the walk is not reading the core", read)
	}
}

// A test file is left out of the rules above, because a test stands outside the
// package it exercises and builds the adapters that stand in for the real ones.
// That leaves the packages holding what is true of a note or a card, where the
// reason does not reach: nothing there needs a disk to be shown a note is what
// it is, and a test that takes one has turned the dependency round at the
// innermost layer, where every other package can see it.
func TestNoPurePackageIsTestedThroughAnAdapter(t *testing.T) {
	var wrong []string
	var read int
	reached := map[string]bool{}
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return err
		}
		pkg := getPackage("..", path)
		if !isAmong(pure, pkg) {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		read++
		for _, one := range pure {
			if pkg == one || strings.HasPrefix(pkg, one+"/") {
				reached[one] = true
			}
		}
		for _, one := range file.Imports {
			to, err := strconv.Unquote(one.Path.Value)
			if err != nil || !strings.HasPrefix(to, module) {
				continue
			}
			target := strings.TrimPrefix(to, module)
			if isAdapter(target) || target == "container" {
				wrong = append(wrong, path+" is tested through "+target)
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

	// A walk that read no test of a pure package is a rule checked against
	// nothing, and it passes. A count cannot say which of them it read, and
	// four of the five would clear any floor the fifth is left out of, so each
	// is named: a package that moved is a package this rule stopped at.
	if read < 5 {
		t.Fatalf("%d tests of the pure packages read: the walk is not reading them", read)
	}
	for _, one := range pure {
		if !reached[one] {
			t.Errorf("%s has no test the walk read: the rule stops at that package", one)
		}
	}
}

// A port is named after the need and an adapter after the technology, and the
// binding between them is the composition root's. An adapter that writes
// `var _ port.X = …` puts that binding in two places.
//
// A method returning `port.X` is not that claim. Where a port opens another —
// `VaultReaders.Open` answers with a `VaultReader` — the return type is the
// port's own signature, and an adapter bound to the first has to write the
// second. What this refuses is the free-standing claim.
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

	// A walk that read no file of an adapter is a rule checked against nothing,
	// and it passes. Every adapter there is is named, so the count stands
	// against all of them.
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

// A test that builds an adapter stands outside the package it exercises. That
// is what the rules above are left out of a test file for: an adapter built
// from within is in the package's own compilation unit when the tests build,
// and every other scenario in the core is tested from a package_test.
//
// An adapter's own test is not this. It builds the thing it is about, and so
// does the composition root's.
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

// theMachine are the packages that are the machine and the world beyond it: the
// process's files and its children, a socket, and the kernel itself. Each one
// is an answer a scenario asks a port for.
//
// net/url is not here. An address taken apart and put together again is
// arithmetic over a string, and nothing is dialled to do it.
var theMachine = map[string]bool{
	"os": true, "os/exec": true, "net": true, "net/http": true, "syscall": true,
}

// golang.org/x/sys is the kernel under another name: every package beneath it
// is one operating system's own calls and error numbers.
const systemCalls = "golang.org/x/sys/"

// isTheMachine answers whether an import is the machine.
func isTheMachine(to string) bool { return theMachine[to] || strings.HasPrefix(to, systemCalls) }

// looking are the functions of path/filepath that are not path arithmetic.
// Each one asks the machine what is there, and Abs answers against the folder
// the process was started in, which a scenario is not written against.
var looking = map[string]bool{
	"EvalSymlinks": true, "Glob": true, "Walk": true, "WalkDir": true, "Abs": true,
}

// A scenario asks a port what is on the machine, and never the machine. What a
// path is once every link on the way to it is resolved is one such question,
// and a folder handed in as a content URI has no answer for it: the disk this
// would read is not the one the file is on. A file, a socket and a system call
// are the same question asked further down.
//
// A path taken apart and put together again is arithmetic over a string and
// stays here. What separates the two is whether the answer is on the disk.
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
	"github.com/jiva-studio/numen/modules/libs/protocol",
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

func TestWhatTheTransportRuleRefuses(t *testing.T) {
	var rejected []string
	for _, to := range []string{
		"connectrpc.com/connect",
		"database/sql",
		"database/sql/driver",
		"net/http",
		"net/http/httptest",
		"github.com/jiva-studio/numen/modules/libs/protocol",
		"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1",
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
		"github.com/jiva-studio/numen/modules/libs/protocol",
		"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1",
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

// Nothing here logs. A log is a record kept for somebody who was not there, and
// the person this application is for is in front of the window: what has to be
// said is said there, or on the stream whoever started the process is reading.
// A logger bound anywhere would write to neither.
//
// The rule is the whole core's and not a layer's, so the adapters and the
// composition root are read too.
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
func TestWhatTheLoggingRuleRefuses(t *testing.T) {
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

// telling are the functions of time that answer from the machine's clock, and
// waiting are the ones that arm the machine's timers. The rest of the package
// is arithmetic over an instant somebody was given.
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

// The domain does not know what time it is. A clock is a port, so a scenario
// that stamps a note or asks what is due today is handed one, and the
// composition root is where it is bound to this machine's.
//
// A scenario reading the clock itself cannot be exercised at a chosen instant,
// and scheduling here is entirely about when: the day a card comes back on, the
// budget the day is answered under, and the identifier a note is stamped with
// all turn on it.
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

// An identity the core carries is a word of the core's own. A row number is the
// store's: it says which row of which table, whatever counts rows hands it out,
// and it can be ordered and paged when the thing it names cannot. A core
// carrying one has taken the store's numbering into its language, and then into
// what a scenario does — which is how a use case evaluates to loop on ascending
// integer keys.
//
// Every identity here is its own type, and this is what keeps it so.
func TestTheCoresIdentitiesAreItsOwn(t *testing.T) {
	var wrong []string
	var read int
	for _, item := range []string{"domain", "port"} {
		dir := getPackageDir(item)
		target, err := os.ReadDir(dir)
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
func TestWhatTheIdentityRuleRefuses(t *testing.T) {
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
				if from, is := offset.X.(*ast.Ident); is && from.Name == "port" {
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
func TestWhatThePortClaimRefuses(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "claim.go", `package p

var _ port.VaultReader = (*Reader)(nil)

var _, _ port.VaultWriter = (*Writer)(nil), (*Writer)(nil)

var storePort port.DerivedStore

func mount() {
	var _ port.Recording = (*sound)(nil)
	var agentPort port.Agent
	_, _ = storePort, agentPort
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"VaultReader", "VaultWriter", "Recording"}
	if got := getClaimedPorts(file); !slices.Equal(got, want) {
		t.Errorf("the rule refuses %v, want %v", got, want)
	}
}

// schema is the generated messages. An adapter that takes one and answers with
// one is answering something outside, whether it is served over the wire or
// called by another adapter, and that is what makes it a driving adapter.
//
// The handler it satisfies is not what says so: an adapter never names the
// interface it answers to, so nothing but the messages is left to read.
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

// A port is a conversation the core holds with something outside it, and a
// conversation nothing asks for is not one. An interface left in port/ after
// the last caller went is indirection standing on its own, and the composition
// root goes on binding an adapter to it.
//
// The tests are read: a port a test alone still asks for is asked for.
func TestEveryPortIsAskedForSomewhereElse(t *testing.T) {
	unasked, err := getUnnamedTypes("..", filepath.Join("..", "port"), true)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range unasked {
		t.Errorf("port.%s is declared and nothing asks for it", one)
	}
}

// A port is named in the words of what it is about, and those words are
// declared beside it: the identity of a model, a page of audio, a row of the
// settings file. A word nothing outside port/ says is a word of no
// conversation, and it stays behind when the conversation it belonged to goes.
//
// This does not say which words belong here. Whether a type is the domain's or
// the boundary's is read, not parsed: what the index holds is the domain's, and
// what an adapter is configured with is the boundary's.
func TestEveryTypePortDeclaresIsNamedSomewhereElse(t *testing.T) {
	unnamed, err := getUnnamedTypes("..", filepath.Join("..", "port"), false)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range unnamed {
		t.Errorf("port.%s is declared and nothing outside port/ names it", one)
	}
}

// Read against port/ as the whole tree, nothing outside it names anything, and
// the rule has to come back with the names below. It is what says the passes
// above are passing on the naming and not on an empty walk.
func TestWhatThePortRuleRefuses(t *testing.T) {
	pos := filepath.Join("..", "port")
	unnamed, err := getUnnamedTypes(pos, pos, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range []string{"TextExtractor", "PageRenderer", "Vector", "ErrorHandler"} {
		if !slices.Contains(unnamed, one) {
			t.Errorf("nothing outside names port.%s and the rule does not refuse it", one)
		}
	}
}

// A port is asked for as a whole, and a method of it is one thing the core can
// ask. A method nothing calls is a question never put: the adapters go on
// answering it, and the day it is wrong nothing says so.
//
// The tests are not read here, and that asymmetry is the rule. A port a test
// alone asks for is still a conversation, so the two rules above read them. A
// method a test alone calls is the opposite: the only caller is the thing that
// was written to check the answer, so the method exists to be tested and
// nothing else is asking.
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
func TestWhatThePortMethodRuleRefuses(t *testing.T) {
	pos := filepath.Join("..", "port")
	uncalled, err := getUncalledMethods(pos, pos)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range []string{"NoteQueries.GetNameMatches", "Transcriber.GetTranscriptionModel"} {
		if !slices.Contains(uncalled, one) {
			t.Errorf("nothing outside calls %s and the rule does not refuse it", one)
		}
	}
}

// getUncalledMethods are the methods port/'s interfaces declare that no file of
// the tree outside it, and outside a test, names as a selector.
//
// A selector is what a call site of a method looks like whatever holds the
// value, so the match is on the name alone: an interface the core reaches
// through a variable never says port.X at the call site.
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
		if strings.HasSuffix(path, "_test.go") || filepath.Dir(path) == filepath.Clean(dir) {
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
	target, err := os.ReadDir(dir)
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
		if getPackage(root, path) == "port" {
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
			if from, is := pos.X.(*ast.Ident); is && from.Name == "port" {
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

// getDeclaredTypes are the exported types a folder's own files declare, each
// said to be an interface or not.
func getDeclaredTypes(dir string) (map[string]bool, error) {
	target, err := os.ReadDir(dir)
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

// TestWhatTheLayerAliasRuleRefuses verifies rejection of layer names as import aliases.
func TestWhatTheLayerAliasRuleRefuses(t *testing.T) {
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

// getRefusal says why one package may not reach another, and nothing where it
// may.
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
		if isAmong(pure, from) && !isAmong(pure, to) {
			return "what is true of a note or a card is worked out from what is too"
		}
	}
	return ""
}

// isAmong says whether a package is one of these, or is placed under one.
func isAmong(these []string, pkg string) bool {
	for _, one := range these {
		if pkg == one || strings.HasPrefix(pkg, one+"/") {
			return true
		}
	}
	return false
}

// isAssembled says whether a package is one the composition root puts together:
// an adapter, a scenario, and the two languages the two are named in.
func isAssembled(to string) bool {
	return to == "container" || to == "domain" || to == "port" ||
		isAdapter(to) || strings.HasPrefix(to, "usecase/")
}

// sibling says whether two packages are one adapter: its own folder and every
// package under it, which share the settings section they are built from.
func sibling(from, to string) bool { return family(from) == family(to) }

// grouping are the folders that hold adapters and are none themselves: adapter/
// itself, the folder the compiler holds them in, and the folder holding one
// adapter to a window. What stands directly under any of them is an adapter.
var grouping = []string{"adapter", "internal/adapter", "adapter/window", "internal/adapterlib"}

// family is the adapter a package belongs to: what stands directly under the
// grouping folder it is in, and every package beneath that.
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
	"window/editor", "window/flashcards",
}

// The core's surface is these adapters and no others. A driven adapter nothing
// outside composes sits under internal/adapter, where the compiler holds it, so
// binding it to a port stays this package's work.
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
	"appstate", "download", "filesystem", "keychain", "onnxruntime", "plugin",
	"theme", "trash",
}

// The adapters under internal/ are these and no others.
func TestTheCoresHeldAdaptersAreTheseAndNoOthers(t *testing.T) {
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
		{"adapter/window/editor", "adapter/window/flashcards", true},
		{"adapter/window/flashcards", "adapter/window/editor", true},
		{"adapter/window/editor/pages", "adapter/window/editor", false},

		// A driven adapter runs no scenario, and takes no other adapter. Which
		// of the two an adapter is has nothing to do with the folder it sits
		// in: the one the compiler holds here serves the schema.
		{"adapter/index", "usecase/note", true},
		{"internal/adapter/filesystem", "usecase/note", true},
		{"internal/adapter/theme", "usecase/note", false},
		{"internal/adapter/theme", "internal/adapter/filesystem", true},
		{"internal/adapter/theme/presets", "internal/adapter/theme", false},
		{"adapter/index/chunk", "adapter/index", false},

		// The core reaches no adapter and nothing that assembles one.
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

		// What is true of a note is worked out from what is true of a note.
		{"internal/highlight", "internal/epub", true},
		{"internal/ocr", "internal/highlight", false},
		{"internal/transcript", "port", true},

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

// An instant the core carries is a time.Time. A second count and a millisecond
// count are both integers, so one written where the other is read compiles and
// is wrong by a factor of a thousand; the zero of either is a real instant in
// 1970, and the clock it was read from is nowhere in the name.
//
// The rule holds over domain/ and port/, which is where the core says what it
// is about and what it asks of the outside.
func TestEveryInstantTheCoreCarriesIsATime(t *testing.T) {
	var wrong []string
	var read int
	for _, item := range []string{"domain", "port"} {
		dir := getPackageDir(item)
		target, err := os.ReadDir(dir)
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
func TestWhatTheInstantRuleRefuses(t *testing.T) {
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

// The same rule over what a port asks for. An identity handed across a port as
// a number is the store's numbering crossing into the language the core is
// written in, and a signature is where it crosses.
func TestTheIdentitiesAPortAsksForAreTheCoresOwn(t *testing.T) {
	var wrong []string
	var read int
	dir := filepath.Join("..", "port")
	target, err := os.ReadDir(dir)
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

// reaches a parameter and a result alike, lets through the one that is the
// core's own word, and reads nothing into a parameter with no name.
func TestWhatTheIdentityRuleRefusesInASignature(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "asking.go", `package p

import "context"

type Store interface {
	Read(ctx context.Context, vaultID domain.VaultID, chunkID int64) error
	Write(ctx context.Context, rowID int) (cardID domain.CardID, err error)
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
func TestWhatTheClockRuleRefuses(t *testing.T) {
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
