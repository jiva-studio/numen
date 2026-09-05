package container

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// module is what every package of the core is named under.
const module = "github.com/jiva-studio/numen/modules/libs/core/"

// owed are the edges this installation still has. Each is a package reaching a
// sibling it should be given instead, and the list only shrinks.
//
// A package is named by the path it sits at. An adapter the compiler holds sits
// under internal/, and a name here is not a name there: what is owed to
// adapter/webui is not owed to internal/adapter/webui, which is not the same
// package and would not be the same window.
var owed = map[string][]string{
	// The terminal opens a vault and reads the settings file itself.
	"adapter/cli": {"internal/adapter/filesystem", "adapter/settings", "container"},
	// The window assembles what it serves.
	"adapter/webui": {"container"},
	// One settings file is the union of every adapter's section.
	"adapter/settings": {
		"adapter/agent", "internal/adapter/embed", "internal/adapter/proofreading",
		"internal/adapter/recognition", "internal/adapter/transcription",
	},
	// The two source queues and the deck writer stand here, so the words for a
	// piece of work, a card, a schedule, a cut and a vector are read in place.
	"container": {
		"chunking", "embedding", "flashcards/format", "flashcards/review",
		"markdown", "proofread", "task",
	},
}

// driving are the adapters something outside comes in through. They call the
// scenarios; a driven adapter stands behind a port and calls none, so what a
// scenario is written over is a port and never an adapter's own answer.
//
// Which way an adapter faces has nothing to do with where it stands. internal/
// says nothing outside composes this, and a driving adapter mounted by another
// adapter rather than by an application is composed by nothing outside.
var driving = map[string]bool{
	"adapter/cli":            true,
	"adapter/flashcardsui":   true,
	"adapter/mcp":            true,
	"adapter/webui":          true,
	"internal/adapter/theme": true,
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
		from := within(root, path)
		for _, one := range file.Imports {
			to, err := strconv.Unquote(one.Path.Value)
			if err != nil || !strings.HasPrefix(to, module) {
				continue
			}
			read++
			if why := refused(from, strings.TrimPrefix(to, module)); why != "" {
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
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return err
		}
		if !holds(pure, within("..", path)) {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, one := range file.Imports {
			to, err := strconv.Unquote(one.Path.Value)
			if err != nil || !strings.HasPrefix(to, module) {
				continue
			}
			held := strings.TrimPrefix(to, module)
			if adapting(held) || held == "container" {
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
}

// A port is named after the need and an adapter after the technology, and the
// binding between them is the composition root's. An adapter that writes
// `var _ port.X = …` has named the need it answers, which puts the binding in
// two places: the day the port grows a method, the adapter fails to compile
// where nothing yet asks it for that method.
func TestNoAdapterNamesThePortItSatisfies(t *testing.T) {
	var wrong []string
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		if strings.HasSuffix(path, "_test.go") || !adapting(within("..", path)) {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		for _, one := range claimed(file) {
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
}

// The settings file is the union of every adapter's section, which is what the
// edges out of adapter/settings are owed for. A section is another adapter's
// shape and the defaults it starts at, and never its work: an adapter that
// stats a folder on another's behalf is doing the work that other one is bound
// for, in a package the composition root binds nothing of.
func TestTheSettingsAdapterRunsNoOtherAdaptersWork(t *testing.T) {
	at := filepath.Join("..", "adapter", "settings")
	held, err := os.ReadDir(at)
	if err != nil {
		t.Fatal(err)
	}
	var wrong []string
	var read int
	for _, one := range held {
		if one.IsDir() || !strings.HasSuffix(one.Name(), ".go") ||
			strings.HasSuffix(one.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(at, one.Name()), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		named := adapters(file, "adapter/settings")
		ast.Inspect(file, func(node ast.Node) bool {
			call, is := node.(*ast.CallExpr)
			if !is {
				return true
			}
			ran, is := call.Fun.(*ast.SelectorExpr)
			if !is {
				return true
			}
			from, is := ran.X.(*ast.Ident)
			if !is || !named[from.Name] {
				return true
			}
			read++
			if !strings.HasSuffix(ran.Sel.Name, "Defaults") {
				wrong = append(wrong, one.Name()+" runs "+from.Name+"."+ran.Sel.Name)
			}
			return true
		})
	}
	for _, one := range wrong {
		t.Error(one + ": a settings section is another adapter's shape and its defaults")
	}

	// A walk that read no call into another adapter is a rule checked against
	// nothing, and it passes.
	if read == 0 {
		t.Fatal("adapter/settings names no other adapter: the walk is not reading it")
	}
}

// adapters are the names one file calls another adapter's package by, whether
// that is the package's own name or an alias.
func adapters(file *ast.File, own string) map[string]bool {
	named := map[string]bool{}
	for _, one := range file.Imports {
		to, err := strconv.Unquote(one.Path.Value)
		if err != nil || !strings.HasPrefix(to, module) {
			continue
		}
		held := strings.TrimPrefix(to, module)
		if !adapting(held) || family(held) == own {
			continue
		}
		name := held[strings.LastIndex(held, "/")+1:]
		if one.Name != nil {
			name = one.Name.Name
		}
		named[name] = true
	}
	return named
}

// claimed are the ports a file declares itself to answer, by the blank name.
func claimed(file *ast.File) []string {
	var held []string
	for _, one := range file.Decls {
		decl, is := one.(*ast.GenDecl)
		if !is || decl.Tok != token.VAR {
			continue
		}
		for _, spec := range decl.Specs {
			named, is := spec.(*ast.ValueSpec)
			if !is || len(named.Names) != 1 || named.Names[0].Name != "_" {
				continue
			}
			if at, is := named.Type.(*ast.SelectorExpr); is {
				if from, is := at.X.(*ast.Ident); is && from.Name == "port" {
					held = append(held, at.Sel.Name)
				}
			}
		}
	}
	return held
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
		pkg := within("..", path)
		if strings.HasSuffix(path, "_test.go") || !adapting(pkg) {
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
	declared, err := interfaces(filepath.Join("..", "port"))
	if err != nil {
		t.Fatal(err)
	}
	if len(declared) == 0 {
		t.Fatal("port/ declares no interface: the walk is not reading it")
	}

	asked := map[string]bool{}
	err = filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		if within("..", path) == "port" {
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
				asked[at.Sel.Name] = true
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, one := range declared {
		if !asked[one] {
			t.Errorf("port.%s is declared and nothing asks for it", one)
		}
	}
}

// interfaces are the exported interfaces a folder's own files declare.
func interfaces(dir string) ([]string, error) {
	held, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var declared []string
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
				if _, is := named.Type.(*ast.InterfaceType); is {
					declared = append(declared, named.Name.Name)
				}
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
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, one := range file.Imports {
			if one.Name == nil || !layers[one.Name.Name] {
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
}

// within is the package a file belongs to, as the rules name it.
func within(root, path string) string {
	held, err := filepath.Rel(root, filepath.Dir(path))
	if err != nil {
		return path
	}
	return filepath.ToSlash(held)
}

// adapting says whether a package is an adapter, wherever it stands. An adapter
// the compiler holds sits under internal/ and is held to the same rules.
func adapting(pkg string) bool {
	return strings.HasPrefix(pkg, "adapter/") ||
		strings.HasPrefix(pkg, "internal/adapter/")
}

// refused says why one package may not reach another, and nothing where it may.
func refused(from, to string) string {
	for _, held := range owed[from] {
		if to == held || strings.HasPrefix(to, held+"/") {
			return ""
		}
	}

	switch {
	case adapting(from):
		if to == "container" {
			return "an adapter is given what it needs and assembles nothing"
		}
		if adapting(to) && !sibling(from, to) {
			return "an adapter is given what it needs and takes no other adapter"
		}
		if strings.HasPrefix(to, "usecase/") && !driving[family(from)] {
			return "a driven adapter stands behind a port and runs no scenario"
		}
	case from == "container":
		if !assembling(to) {
			return "the composition root assembles the core and does none of its work"
		}
	// Fixtures build the real adapters, and only a test is compiled from them.
	case holds([]string{"internal/testsupport", "internal/testonly"}, from):
	default:
		if adapting(to) || to == "container" {
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

// assembling says whether a package is one the composition root puts together:
// an adapter, a scenario, and the two languages the two are named in.
func assembling(to string) bool {
	return to == "container" || to == "domain" || to == "port" ||
		adapting(to) || strings.HasPrefix(to, "usecase/")
}

// sibling says whether two packages are one adapter: its own folder and every
// package under it, which share the settings section they are built from.
func sibling(from, to string) bool { return family(from) == family(to) }

// family is the adapter a package belongs to: the folder holding it, and where
// the compiler holds that folder, the folder under internal/adapter.
func family(pkg string) string {
	held := strings.Split(pkg, "/")
	depth := 2
	if strings.HasPrefix(pkg, "internal/adapter/") {
		depth = 3
	}
	if len(held) < depth {
		return pkg
	}
	return strings.Join(held[:depth], "/")
}

// public are the adapters an application names for itself: the four it serves
// something through, and the three it composes or configures.
var public = []string{
	"agent", "cli", "flashcardsui", "index", "mcp", "settings", "webui",
}

// The core's surface is these adapters and no others. A driven adapter nothing
// outside composes sits under internal/adapter, where the compiler holds it, so
// binding it to a port stays this package's work.
func TestTheCoresPublicAdaptersAreTheseAndNoOthers(t *testing.T) {
	held, err := os.ReadDir(filepath.Join("..", "adapter"))
	if err != nil {
		t.Fatal(err)
	}
	standing := make(map[string]bool, len(held))
	for _, one := range held {
		if !one.IsDir() {
			continue
		}
		standing[one.Name()] = true
		if !holds(public, one.Name()) {
			t.Errorf("adapter/%s is public and nothing outside composes it", one.Name())
		}
	}
	for _, one := range public {
		if !standing[one] {
			t.Errorf("adapter/%s is named here and is not there", one)
		}
	}
}

// held are the adapters the compiler keeps to this module. Naming them is what
// makes a new one arrive as a decision: the folder is otherwise unread, and an
// adapter put there is bound to a port by nobody and noticed by nothing.
var held = []string{
	"appstate", "embed", "filesystem", "pdf", "proofreading",
	"recognition", "theme", "transcription", "trash",
}

// The adapters under internal/ are these and no others.
func TestTheCoresHeldAdaptersAreTheseAndNoOthers(t *testing.T) {
	found, err := os.ReadDir(filepath.Join("..", "internal", "adapter"))
	if err != nil {
		t.Fatal(err)
	}
	standing := make(map[string]bool, len(found))
	for _, one := range found {
		if !one.IsDir() {
			continue
		}
		standing[one.Name()] = true
		if !holds(held, one.Name()) {
			t.Errorf("internal/adapter/%s stands here and is named nowhere", one.Name())
		}
	}
	for _, one := range held {
		if !standing[one] {
			t.Errorf("internal/adapter/%s is named here and is not there", one)
		}
	}
}

// What refused answers is the whole of the rule, so it is asked directly. Each
// refusal here is an edge that was walked around once: an adapter the compiler
// holds took the name of one it does not and was given what that name is owed,
// and a package holding what is true of a note reached one that holds work.
func TestWhatTheRulesRefuse(t *testing.T) {
	for _, one := range []struct {
		from, to string
		refuses  bool
	}{
		// An adapter under internal/ is not the adapter it is named after.
		{"internal/adapter/webui", "container", true},
		{"internal/adapter/cli", "internal/adapter/filesystem", true},
		{"internal/adapter/mcp", "usecase/note", true},
		{"adapter/webui", "container", false},
		{"adapter/cli", "internal/adapter/filesystem", false},
		{"adapter/mcp", "usecase/note", false},

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
		{"internal/wire", "adapter/webui", true},

		// What is true of a note is worked out from what is true of a note.
		{"domain", "task", true},
		{"flashcards/review", "port", true},
		{"flashcards/review", "chunking", true},
		{"domain", "markdown", false},
		{"flashcards/format", "domain", false},

		// The composition root does none of the core's work.
		{"container", "internal/wire", true},
		{"container", "text", true},
		{"container", "adapter/index", false},
		{"container", "task", false},
	} {
		why := refused(one.from, one.to)
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
