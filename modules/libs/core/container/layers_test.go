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

// module is what every package of the core is named under.
const module = "github.com/jiva-studio/numen/modules/libs/core/"

// baseline are the edges this installation still has, and the list only
// shrinks.
//
// A package is named by the path it sits at. An adapter the compiler holds sits
// under internal/, and a name here is not a name there: the two are different
// packages and an entry for one admits nothing of the other.
var baseline = map[string][]string{
	// The composition root is compiled from the words the things it binds are
	// named in. It hands a scenario the day a review begins at and a worker the
	// list of what is being done, and it reads off the settings how wide a
	// chunk is cut; the cutting, the scheduling and the work are elsewhere.
	"container": {"internal/chunking", "flashcards/review", "task"},
}

// driving are the adapters something outside comes in through. They call the
// scenarios; a driven adapter stands behind a port and calls none, so what a
// scenario is written over is a port and never an adapter's own answer.
//
// Which way an adapter faces has nothing to do with where it stands: internal/
// says only that nothing outside composes it, which holds of a driving adapter
// another adapter mounts.
var driving = map[string]bool{
	"adapter/cli":               true,
	"adapter/mcp":               true,
	"adapter/window/editor":     true,
	"adapter/window/flashcards": true,
	"internal/adapter/theme":    true,
}

// pure are the packages holding what is true of a note or a card, and the
// arithmetic over it. Nothing they answer waits on a disk, a database or a
// clock, and the way that is held is that a pure package reaches only another
// pure one: naming no port and no scenario is not enough on its own, because a
// sibling of the core carries goroutines, channels and a schema.
var pure = []string{
	"domain", "flashcards", "markdown", "internal/cardid", "internal/ulid",
}

// The core is reached by the adapters and reaches none of them, an adapter is
// given what it needs rather than taking another, and what is assembled is
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
		if !holds(pure, pkg) {
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
			held := strings.TrimPrefix(to, module)
			if isAdapter(held) || held == "container" {
				wrong = append(wrong, path+" is tested through "+held)
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
	if read < 20 {
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
		if strings.HasSuffix(path, "_test.go") || !isAdapter(pkg) {
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
	for _, one := range held {
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
func TestATestOfTheCoreBuildingAnAdapterStandsOutsideIt(t *testing.T) {
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
			if held := strings.TrimPrefix(to, module); isAdapter(held) {
				wrong = append(wrong, path+" builds "+held+" from inside "+pkg)
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

// machinery are the packages held to none of the abstinences below, because
// reaching outward is what they are: the composition root that builds the
// adapters, the fixtures only a test is compiled from, and the infrastructure
// the adapters share — the runtime two of them load their models through, and
// the package two windows both answer a request for a page out of. The
// adapters themselves are left out by adapting.
var machinery = []string{
	"container", "internal/onnxruntime", "internal/testsupport", "internal/wire",
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
		if strings.HasSuffix(path, "_test.go") || isAdapter(pkg) || holds(machinery, pkg) {
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
			at, is := node.(*ast.SelectorExpr)
			if !is {
				return true
			}
			from, is := at.X.(*ast.Ident)
			if is && from.Name == "filepath" && looking[at.Sel.Name] {
				wrong = append(wrong, path+" runs filepath."+at.Sel.Name)
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

// What the rule refuses, read against imports written to be refused. It has to
// find both of the standard library's and a third party's, and let through the
// packages whose names only begin the same way.
func TestWhatTheLoggingRuleRefuses(t *testing.T) {
	var refused []string
	for _, to := range []string{
		"log", "log/slog", "go.uber.org/zap", "github.com/rs/zerolog",
		"logic", "text/template", module + "domain",
	} {
		if logs(to) {
			refused = append(refused, to)
		}
	}
	want := []string{"log", "log/slog", "go.uber.org/zap", "github.com/rs/zerolog"}
	if !slices.Equal(refused, want) {
		t.Errorf("the rule refuses %v, want %v", refused, want)
	}
}

// telling are the functions of time that answer from the machine's clock. The
// rest of the package is arithmetic over an instant somebody was given.
var telling = map[string]bool{"Now": true, "Since": true}

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
		if strings.HasSuffix(path, "_test.go") || isAdapter(pkg) || holds(machinery, pkg) {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		read++
		ast.Inspect(file, func(node ast.Node) bool {
			at, is := node.(*ast.SelectorExpr)
			if !is {
				return true
			}
			from, is := at.X.(*ast.Ident)
			if is && from.Name == "time" && telling[at.Sel.Name] {
				wrong = append(wrong, path+" runs time."+at.Sel.Name)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one + ": a scenario is handed a clock and reads no other")
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

// identity is an exported field naming one: the type it stands in, the field,
// and how its own type is written.
type identity struct{ in, field, spelled string }

// identities are the exported struct fields of one file whose names end in ID.
func identities(file *ast.File) []identity {
	var found []identity
	ast.Inspect(file, func(node ast.Node) bool {
		spec, is := node.(*ast.TypeSpec)
		if !is {
			return true
		}
		held, is := spec.Type.(*ast.StructType)
		if !is {
			return true
		}
		for _, one := range held.Fields.List {
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
func getTypeName(at ast.Expr) string {
	switch held := at.(type) {
	case *ast.Ident:
		return held.Name
	case *ast.SelectorExpr:
		if from, is := held.X.(*ast.Ident); is {
			return from.Name + "." + held.Sel.Name
		}
	}
	return ""
}

// An identity the core carries is a word of the core's own. A row number is the
// store's: it says which row of which table, whatever counts rows hands it out,
// and it can be ordered and paged when the thing it names cannot. A core
// carrying one has taken the store's numbering into its language, and then into
// what a scenario does — which is how a use case comes to loop on ascending
// integer keys.
//
// Every identity here is its own type, and this is what keeps it so.
func TestTheCoresIdentitiesAreItsOwn(t *testing.T) {
	var wrong []string
	var read int
	for _, at := range []string{"domain", "port"} {
		dir := filepath.Join("..", at)
		held, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, one := range held {
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
						at, found.in, found.field, found.spelled))
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

// What the rule refuses, read against a declaration written to be refused. It
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
	var refused []string
	for _, one := range identities(file) {
		if numbered[one.spelled] {
			refused = append(refused, one.field)
		}
	}
	if !slices.Equal(refused, []string{"ChunkID"}) {
		t.Errorf("the rule refuses %v, want the one identity spelled as a number", refused)
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
	var held []string
	ast.Inspect(file, func(node ast.Node) bool {
		decl, is := node.(*ast.GenDecl)
		if !is || decl.Tok != token.VAR {
			return true
		}
		for _, spec := range decl.Specs {
			named, is := spec.(*ast.ValueSpec)
			if !is || !slices.ContainsFunc(named.Names, func(at *ast.Ident) bool {
				return at.Name == "_"
			}) {
				continue
			}
			if at, is := named.Type.(*ast.SelectorExpr); is {
				if from, is := at.X.(*ast.Ident); is && from.Name == "port" {
					held = append(held, at.Sel.Name)
				}
			}
		}
		return true
	})
	return held
}

// What the rule refuses, read against a file written to be refused. The three
// shapes of the claim are one claim, and the walk has to reach the one written
// inside a function; a variable holding a port is left alone.
func TestWhatThePortClaimRefuses(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "claim.go", `package p

var _ port.VaultReader = (*Reader)(nil)

var _, _ port.VaultWriter = (*Writer)(nil), (*Writer)(nil)

var held port.DerivedStore

func mount() {
	var _ port.Recording = (*sound)(nil)
	var kept port.Agent
	_, _ = held, kept
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

// Which way an adapter faces is read off what it does, and not off the folder
// it sits in. An adapter that serves or mounts the generated handler is a
// driving adapter wherever it stands, and the list above says so, so the day
// one of them needs a scenario it is allowed one.
func TestEveryAdapterServingTheSchemaIsDriving(t *testing.T) {
	var wrong []string
	var found int
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		pkg := getPackage("..", path)
		if strings.HasSuffix(path, "_test.go") || !isAdapter(pkg) {
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
	at := filepath.Join("..", "port")
	unnamed, err := getUnnamedTypes(at, at, false)
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
	at := filepath.Join("..", "port")
	uncalled, err := getUncalledMethods(at, at)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range []string{"NoteQueries.Names", "Transcriber.Transcription"} {
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
			if at, is := node.(*ast.SelectorExpr); is {
				called[at.Sel.Name] = true
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
	held, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	declared := map[string]string{}
	for _, one := range held {
		if one.IsDir() || !strings.HasSuffix(one.Name(), ".go") ||
			strings.HasSuffix(one.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, one.Name()), nil, 0)
		if err != nil {
			return nil, err
		}
		for _, decl := range file.Decls {
			at, is := decl.(*ast.GenDecl)
			if !is || at.Tok != token.TYPE {
				continue
			}
			for _, spec := range at.Specs {
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
					for _, at := range method.Names {
						if at.IsExported() {
							declared[named.Name.Name+"."+at.Name] = at.Name
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
			at, is := node.(*ast.SelectorExpr)
			if !is {
				return true
			}
			if from, is := at.X.(*ast.Ident); is && from.Name == "port" {
				named[at.Sel.Name] = true
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
	held, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	declared := map[string]bool{}
	for _, one := range held {
		if one.IsDir() || !strings.HasSuffix(one.Name(), ".go") ||
			strings.HasSuffix(one.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, one.Name()), nil, 0)
		if err != nil {
			return nil, err
		}
		for _, decl := range file.Decls {
			at, is := decl.(*ast.GenDecl)
			if !is || at.Tok != token.TYPE {
				continue
			}
			for _, spec := range at.Specs {
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

// A package is named at the call site for what it is. An alias naming a layer
// says only which folder the package sits in, which the import path already
// says, and the reader is left with usecase.Add for what is vaults.Add.
//
// The tests are read too: most of a package's call sites are in them.
func TestNoPackageIsImportedUnderItsLayer(t *testing.T) {
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
			held, err := strconv.Unquote(one.Path.Value)
			if err != nil {
				return err
			}
			wrong = append(wrong, path+" names "+held+" "+one.Name.Name)
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

// What the rule refuses, read against imports written to be refused. The name
// of a layer says only which folder the package sits in; any other word is the
// caller saying what the package is, which is the whole point of writing one.
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
	var refused []string
	for _, one := range file.Imports {
		if one.Name != nil && layers[one.Name.Name] {
			refused = append(refused, one.Name.Name)
		}
	}
	if want := []string{"usecase", "adapter"}; !slices.Equal(refused, want) {
		t.Errorf("the rule refuses %v, want %v", refused, want)
	}
}

// getPackage is the package a file belongs to, as the rules name it.
func getPackage(root, path string) string {
	held, err := filepath.Rel(root, filepath.Dir(path))
	if err != nil {
		return path
	}
	return filepath.ToSlash(held)
}

// isAdapter says whether a package is an adapter, wherever it stands. An
// adapter the compiler holds sits under internal/ and is held to the same
// rules.
func isAdapter(pkg string) bool {
	return strings.HasPrefix(pkg, "adapter/") ||
		strings.HasPrefix(pkg, "internal/adapter/")
}

// getRefusal says why one package may not reach another, and nothing where it
// may.
func getRefusal(from, to string) string {
	for _, held := range baseline[from] {
		if to == held || strings.HasPrefix(to, held+"/") {
			return ""
		}
	}

	switch {
	case isAdapter(from):
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
	case holds([]string{"internal/testsupport", "internal/testonly"}, from):
	default:
		if isAdapter(to) || to == "container" {
			return "the core reaches no adapter and nothing that assembles one"
		}
		if holds(pure, from) && !holds(pure, to) {
			return "what is true of a note or a card is worked out from what is too"
		}
	}
	return ""
}

// holds says whether a package is one of these, or stands under one.
func holds(these []string, pkg string) bool {
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
var grouping = []string{"adapter", "internal/adapter", "adapter/window"}

// family is the adapter a package belongs to: what stands directly under the
// grouping folder it is in, and every package beneath that.
func family(pkg string) string {
	depth := 2
	for _, one := range grouping {
		if strings.HasPrefix(pkg, one+"/") {
			depth = max(depth, strings.Count(one, "/")+2)
		}
	}
	held := strings.Split(pkg, "/")
	if len(held) < depth {
		return pkg
	}
	return strings.Join(held[:depth], "/")
}

// getAdaptersIn are the adapters in one folder, named under it. A folder that
// groups adapters is not an adapter, so what stands under it is read in its
// place.
func getAdaptersIn(dir, at string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var found []string
	for _, one := range entries {
		if !one.IsDir() {
			continue
		}
		below := at + "/" + one.Name()
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
// something through, and the three it composes or configures.
var public = []string{
	"agent", "cli", "index", "mcp", "settings",
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
		if !holds(public, one) {
			t.Errorf("adapter/%s is public and nothing outside composes it", one)
		}
	}
	for _, one := range public {
		if !there[one] {
			t.Errorf("adapter/%s is named here and is not there", one)
		}
	}
}

// held are the adapters the compiler keeps to this module. Naming them is what
// makes a new one arrive as a decision.
var held = []string{
	"appstate", "download", "embed", "filesystem", "pdf", "proofreading",
	"recognition", "theme", "transcription", "trash",
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
		if !holds(held, one) {
			t.Errorf("internal/adapter/%s stands here and is named nowhere", one)
		}
	}
	for _, one := range held {
		if !there[one] {
			t.Errorf("internal/adapter/%s is named here and is not there", one)
		}
	}
}

// What refused answers is the whole of the rule, so it is asked directly. Each
// refusal here is an edge that was walked around once: an adapter the compiler
// holds took the name of one it does not and was given that name's baseline
// entry, and a package holding what is true of a note reached one that holds
// work.
func TestWhatTheRulesRefuse(t *testing.T) {
	for _, one := range []struct {
		from, to string
		refuses  bool
	}{
		// An adapter under internal/ is not the adapter it is named after.
		{"internal/adapter/mcp", "usecase/note", true},
		{"adapter/mcp", "usecase/note", false},

		// The window is given what it needs and assembles nothing, and one
		// settings file is the union of sections each adapter owns.
		{"adapter/window/editor", "container", true},
		{"adapter/settings", "internal/adapter/embed", true},

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
		{"internal/wire", "adapter/window/editor", true},

		// What is true of a note is worked out from what is true of a note.
		{"domain", "task", true},
		{"flashcards/review", "port", true},
		{"flashcards/review", "internal/chunking", true},
		{"domain", "markdown", false},
		{"flashcards/format", "domain", false},

		// The composition root does none of the core's work.
		{"container", "internal/wire", true},
		{"container", "internal/text", true},
		{"container", "adapter/index", false},
		{"container", "task", false},
	} {
		why := getRefusal(one.from, one.to)
		if one.refuses && why == "" {
			t.Errorf("%s reaches %s and is not refused", one.from, one.to)
		}
		if !one.refuses && why != "" {
			t.Errorf("%s reaches %s and is refused: %s", one.from, one.to, why)
		}
	}
}

// Reading the tree is what the rules are checked against, so the tree has to be
// where this expects it.
func TestTheTreeIsWhereTheLayersAreRead(t *testing.T) {
	for _, at := range []string{"domain", "port", "usecase", "adapter", "internal/adapter"} {
		if _, err := os.Stat(filepath.Join("..", at)); err != nil {
			t.Fatalf("%s: %v", at, err)
		}
	}
}
